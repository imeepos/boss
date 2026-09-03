package tl1

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ymm-001/boss/internal/domain/provision"
)

// StageEvent 常量,对齐契约 provision/v1 环节事件。
const (
	StagePreConfigOLT     = "preConfigOLT"
	StageActivateUser     = "activateUser"
	StageNotifyActivation = "notifyActivation"
)

// ONU 状态判定用枚举值(PDF §15.4.4)。
const (
	operUp     = "UP"
	cfgNormal  = "NORMAL"
	cfgConfig  = "CONFIG"
	idTypeLOID = "LOID"
)

// Endpoint 网管端点(resolver 取自 provision_nms;连接由调用方经 Manager 建立)。
type Endpoint struct {
	Host       string
	Port       int
	User, Pass string
}

// Params 一次下发任务的全部参数(resolver 一次取齐,缺项由 resolver 报错)。
type Params struct {
	Endpoint        Endpoint
	OLTID, PONID    string // PONID 形如 NA-0-7-5
	AuthType, ONUID string // 缺省 LOID + lo_accounts.loid
	ONUNo           int    // 0~127
	ONUType, Desc   string // Desc=PRV-<orderNo>,幂等对账键基值
	Services        []PONVLANParams
}

// ParamResolver 参数解析口;实现见 internal/app(T5 装配),此处仅契约。
type ParamResolver interface {
	Resolve(ctx context.Context, t provision.Task) (Params, error)
}

// Executor provision.Executor 的 TL1 实现:按 StageEvent 编排先查后写。
type Executor struct {
	m        *Manager
	resolve  ParamResolver
	endpoint Endpoint
}

// NewExecutor 构造;端点由 resolver 返回后按需切换。
func NewExecutor(r ParamResolver) *Executor {
	return &Executor{resolve: r, m: NewManager(Config{})}
}

func (e *Executor) useEndpoint(p Params) {
	if e.endpoint != p.Endpoint {
		e.m.Use(p.Endpoint)
		e.endpoint = p.Endpoint
	}
}

// Exec 按 StageEvent 分派;任一步失败返回 error,由 Daemon 落 FAILED 留痕;
// ExecTrace 带全部 TL1 指令与设备原始应答(后台日志详情页展示)。
func (e *Executor) Exec(ctx context.Context, t provision.Task) (provision.ExecTrace, error) {
	trace := provision.ExecTrace{Commands: []string{}, Driver: provision.DriverTL1}
	rec := &traceSink{trace: &trace}
	p, rerr := e.resolve.Resolve(ctx, t)
	if rerr != nil {
		return trace, e.fail(t, rerr)
	}
	e.useEndpoint(p)
	var err error
	switch t.StageEvent {
	case StagePreConfigOLT:
		err = e.segment(ctx, t, rec, func(d CmdSink) error { return applyPreConfig(ctx, d, t, p) })
	case StageActivateUser:
		err = e.segment(ctx, t, rec, func(d CmdSink) error { return checkActivation(ctx, d, p) })
	case StageNotifyActivation:
		err = e.segment(ctx, t, rec, func(d CmdSink) error {
			for _, svc := range p.Services {
				want := svcDesc(p, svc)
				rows, lerr := lstPONVLAN(ctx, d, p.OLTID, p.PONID, svc.ONUIDType, svc.ONUID)
				if lerr != nil {
					return lerr
				}
				if !descHit(rows, want) {
					return fmt.Errorf("service port missing: %s", want)
				}
			}
			return nil
		})
	default:
		err = fmt.Errorf("unsupported stage event %q", t.StageEvent)
	}
	return trace, e.fail(t, err)
}

// traceSink 记录型命令口:包装真实 sink,逐条留指令与应答原始报文。
type traceSink struct {
	inner CmdSink
	trace *provision.ExecTrace
	seq   int
}

func (s *traceSink) Do(ctx context.Context, c Command) (*Response, error) {
	s.seq++
	r, err := s.inner.Do(ctx, c)
	// 留痕 ctag 与线上对齐:优先取响应实际 ctag(业务 Tag 或会话自增);
	// 无应答(断线/构建失败)才退回 TRC<seq> 占位,不留 B 位假象。
	tag := "TRC" + strconv.Itoa(s.seq)
	if r != nil && r.CTag != "" {
		tag = r.CTag
	}
	line, _ := Build(c, tag)
	s.trace.Commands = append(s.trace.Commands, string(line))
	if r != nil && r.Raw != "" {
		s.trace.Response = joinResp(s.trace.Response, strings.TrimRight(r.Raw, "\n"))
	} else if err != nil {
		s.trace.Response = joinResp(s.trace.Response, err.Error())
	}
	return r, err
}

// joinResp 追加一段应答,分隔行隔开多次交互。
func joinResp(prev, add string) string {
	if prev == "" {
		return add
	}
	return prev + "\n----\n" + add
}

// applyPreConfig 环节7:建 ONU(幂等)+ 逐业务建流(幂等)。
func applyPreConfig(ctx context.Context, d CmdSink, t provision.Task, p Params) error {
	rows, err := lstONU(ctx, d, p.OLTID, p.PONID, idTypeLOID, p.ONUID)
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		log.Printf("[provision-tl1] IDEMPOTENT SKIP(ONU-EXISTS) task=%s onuid=%s", t.TaskNo, p.ONUID)
	} else if err := addONU(ctx, d, p.addONUParams()); err != nil {
		return err
	}
	return applyServices(ctx, d, t, p)
}

// applyServices 逐业务流先查后写;DESC 命中即 SKIP,未命中补建。
func applyServices(ctx context.Context, d CmdSink, t provision.Task, p Params) error {
	for _, svc := range p.Services {
		want := svcDesc(p, svc)
		rows, err := lstPONVLAN(ctx, d, p.OLTID, p.PONID, svc.ONUIDType, svc.ONUID)
		if err != nil {
			return err
		}
		if descHit(rows, want) {
			log.Printf("[provision-tl1] IDEMPOTENT SKIP(PONVLAN-EXISTS) task=%s desc=%s", t.TaskNo, want)
			continue
		}
		w := svc
		w.Name = want // DESC 落网元为完整对账键;ServiceName 原样进 ctag 位
		if err := addPONVLAN(ctx, d, w); err != nil {
			return err
		}
	}
	return nil
}

// checkActivation 环节10:在线且配置就绪才放行,不伪造成功。
func checkActivation(ctx context.Context, d CmdSink, p Params) error {
	st, err := lstONUState(ctx, d, p.OLTID, p.PONID, idTypeLOID, p.ONUID)
	if err != nil {
		return err
	}
	if st.OperState == "" {
		return errors.New("onu not provisioned")
	}
	if st.OperState != operUp {
		return fmt.Errorf("onu not online: %s", st.OperState)
	}
	if st.CFGSTAT != cfgNormal && st.CFGSTAT != cfgConfig {
		return fmt.Errorf("onu cfg state not ready: %s", st.CFGSTAT)
	}
	return nil
}

// segment 多命令原子段:整段持锁,断线退避重连后整段重跑(先查后写保证幂等);
// rec 包装真实 session 采集指令/应答留痕(重连后 inner 随新 session 更新)。
func (e *Executor) segment(ctx context.Context, t provision.Task, rec *traceSink, fn func(CmdSink) error) error {
	err := e.m.WithSession(ctx, func(s *Session) error {
		rec.inner = s
		return fn(rec)
	})
	return e.fail(t, err)
}

// fail 失败留痕:CmdError 带 EN/ENDESC/ctag 全量格式,其余按 stage 记录;可 grep。
func (e *Executor) fail(t provision.Task, err error) error {
	if err == nil {
		return nil
	}
	var ce *CmdError
	if errors.As(err, &ce) && ce.EN != 0 {
		log.Printf("[provision-tl1] EXEC FAILED task=%s ctag=%s cmd=%s EN=%d ENDESC=%s",
			t.TaskNo, ce.CTag, ce.Cmd, ce.EN, ce.ENDESC)
		return err
	}
	log.Printf("[provision-tl1] EXEC FAILED task=%s stage=%s err=%v", t.TaskNo, t.StageEvent, err)
	return err
}

func (p Params) addONUParams() AddONUParams {
	return AddONUParams{
		OLTID: p.OLTID, PONID: p.PONID, AuthType: p.AuthType, ONUID: p.ONUID,
		ONUNo: p.ONUNo, ONUType: p.ONUType, Desc: p.Desc,
	}
}

// svcDesc 业务流对账键:Desc 基值-服务名,如 PRV-<orderNo>-Internet。
// 服务名取 ServiceName(与 Name=DESC 键分离,DESC 不再反哺服务名)。
func svcDesc(p Params, svc PONVLANParams) string { return p.Desc + "-" + svc.ServiceName }

// descHit 行集里存在 DESC==desc 的行。
func descHit(rows []map[string]string, desc string) bool {
	for _, r := range rows {
		if r["DESC"] == desc {
			return true
		}
	}
	return false
}
