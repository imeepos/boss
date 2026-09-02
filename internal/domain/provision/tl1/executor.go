package tl1

import (
	"context"
	"errors"
	"fmt"
	"log"

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

// Exec 按 StageEvent 分派;任一步失败返回 error,由 Daemon 落 FAILED 留痕。
func (e *Executor) Exec(ctx context.Context, t provision.Task) error {
	p, err := e.resolve.Resolve(ctx, t)
	if err != nil {
		return e.fail(t, err)
	}
	e.useEndpoint(p)
	switch t.StageEvent {
	case StagePreConfigOLT:
		return e.fail(t, e.preConfigOLT(ctx, t, p))
	case StageActivateUser:
		return e.fail(t, e.activateUser(ctx, t, p))
	case StageNotifyActivation:
		return e.fail(t, e.notifyActivation(ctx, t, p))
	default:
		return e.fail(t, fmt.Errorf("unsupported stage event %q", t.StageEvent))
	}
}

// preConfigOLT 环节7:建 ONU(幂等)+ 逐业务建流(幂等)。
func (e *Executor) preConfigOLT(ctx context.Context, t provision.Task, p Params) error {
	return e.segment(ctx, t, func(d CmdSink) error {
		return applyPreConfig(ctx, d, t, p)
	})
}

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
		w.Name = want // DESC 落网元为完整对账键
		if err := addPONVLAN(ctx, d, w); err != nil {
			return err
		}
	}
	return nil
}

// activateUser 环节10:在线且配置就绪才放行,不伪造成功。
func (e *Executor) activateUser(ctx context.Context, t provision.Task, p Params) error {
	return e.segment(ctx, t, func(d CmdSink) error {
		return checkActivation(ctx, d, p)
	})
}

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

// notifyActivation 环节11:全部业务流 DESC 命中才允许落回调。
func (e *Executor) notifyActivation(ctx context.Context, t provision.Task, p Params) error {
	return e.segment(ctx, t, func(d CmdSink) error {
		for _, svc := range p.Services {
			want := svcDesc(p, svc)
			rows, err := lstPONVLAN(ctx, d, p.OLTID, p.PONID, svc.ONUIDType, svc.ONUID)
			if err != nil {
				return err
			}
			if !descHit(rows, want) {
				return fmt.Errorf("service port missing: %s", want)
			}
		}
		return nil
	})
}

// segment 多命令原子段:整段持锁,断线退避重连后整段重跑(先查后写保证幂等)。
func (e *Executor) segment(ctx context.Context, t provision.Task, fn func(CmdSink) error) error {
	err := e.m.WithSession(ctx, func(s *Session) error { return fn(s) })
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
func svcDesc(p Params, svc PONVLANParams) string { return p.Desc + "-" + svc.Name }

// descHit 行集里存在 DESC==desc 的行。
func descHit(rows []map[string]string, desc string) bool {
	for _, r := range rows {
		if r["DESC"] == desc {
			return true
		}
	}
	return false
}
