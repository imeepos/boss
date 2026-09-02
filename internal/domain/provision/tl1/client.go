package tl1

import (
	"context"
	"fmt"
	"strconv"
)

// CmdSink 命令执行口:*Manager(独立命令路径)与 *Session(WithSession 原子段)
// 均满足,业务命令与 executor 编排共用同一套构建/判定逻辑。
type CmdSink interface {
	Do(ctx context.Context, c Command) (*Response, error)
}

// ONUState LST-ONUSTATE 单行结果(PDF §15.4.4)。
// OperState: UP 在线 / Power-Off 掉电 / LOS 断纤 / other;
// CFGSTAT: INITIAL / NORMAL / FAILED / CONFIG。
type ONUState struct {
	AdminState, OperState, CFGSTAT string
}

// AddONUParams ADD-ONU 业务参数(PDF §12.2.1)。
type AddONUParams struct {
	OLTID, PONID    string
	AuthType, ONUID string
	ONUNo           int
	ONUType, Desc   string
}

// PONVLANParams ADD-PONVLAN 业务参数(PDF §12.1.1)。
// Name 即 DESC 值(业务别名);executor 传入组合后的 PRV-<orderNo>-<svc> 形态。
// SVLAN 仅 HasSVLAN=true 时下发(TR069 单层业务不带)。
type PONVLANParams struct {
	Name, OLTID, PONID string
	ONUIDType, ONUID   string
	SVLAN, CVLAN, UV   int
	SCOS, CCOS         int
	HasSVLAN           bool
}

// Client 业务命令层:参数结构体→Command→Do→判定→类型化错误。
// 写类命令不做业务级重发(非幂等);连接级重连由 Manager 负责。
type Client struct{ m *Manager }

// NewClient 构造业务客户端。
func NewClient(m *Manager) *Client { return &Client{m: m} }

// LstONU 查询 ONU;空结果返回空切片(网元无此 ONU,非通信失败)。
func (c *Client) LstONU(ctx context.Context, oltid, ponid, idType, id string) ([]map[string]string, error) {
	return lstONU(ctx, c.m, oltid, ponid, idType, id)
}

// LstONUState 查询单个 ONU 状态;查无返回零值 ONUState。
func (c *Client) LstONUState(ctx context.Context, oltid, ponid, idType, id string) (ONUState, error) {
	return lstONUState(ctx, c.m, oltid, ponid, idType, id)
}

// LstPONVLAN 查询业务流;空结果返回空切片。
func (c *Client) LstPONVLAN(ctx context.Context, oltid, ponid, idType, id string) ([]map[string]string, error) {
	return lstPONVLAN(ctx, c.m, oltid, ponid, idType, id)
}

// AddONU 增加 ONU;DENY 返回带 EN/ENDESC/ctag 的 *CmdError。
func (c *Client) AddONU(ctx context.Context, p AddONUParams) error { return addONU(ctx, c.m, p) }

// AddPONVLAN 增加业务流。
func (c *Client) AddPONVLAN(ctx context.Context, p PONVLANParams) error {
	return addPONVLAN(ctx, c.m, p)
}

// ---- 包级实现:Client 方法与 executor 的 WithSession 原子段共用 ----

func lstONU(ctx context.Context, d CmdSink, oltid, ponid, idType, id string) ([]map[string]string, error) {
	rows, err := query(ctx, d, Command{Verb: "LST-ONU", Access: accessONU(oltid, ponid, idType, id)})
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]string{}
	}
	return rows, nil
}

func lstPONVLAN(ctx context.Context, d CmdSink, oltid, ponid, idType, id string) ([]map[string]string, error) {
	rows, err := query(ctx, d, Command{Verb: "LST-PONVLAN", Access: accessONU(oltid, ponid, idType, id)})
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []map[string]string{}
	}
	return rows, nil
}

// lstONUState SHOWOPTION=CFGSTAT 使响应携带 CFGSTAT 列(PDF §15.4.4)。
func lstONUState(ctx context.Context, d CmdSink, oltid, ponid, idType, id string) (ONUState, error) {
	rows, err := query(ctx, d, Command{
		Verb:    "LST-ONUSTATE",
		Access:  accessONU(oltid, ponid, idType, id),
		Payload: []KV{{"SHOWOPTION", "CFGSTAT"}},
	})
	if err != nil || len(rows) == 0 {
		return ONUState{}, err
	}
	r := rows[0]
	return ONUState{AdminState: r["AdminState"], OperState: r["OperState"], CFGSTAT: r["CFGSTAT"]}, nil
}

func addONU(ctx context.Context, d CmdSink, p AddONUParams) error {
	return writeCmd(ctx, d, Command{
		Verb:   "ADD-ONU",
		Access: []KV{{"OLTID", p.OLTID}, {"PONID", p.PONID}},
		Payload: []KV{
			{"AUTHTYPE", p.AuthType}, {"ONUID", p.ONUID},
			{"ONUNO", strconv.Itoa(p.ONUNo)}, {"DESC", p.Desc}, {"ONUTYPE", p.ONUType},
		},
	})
}

// addPONVLAN 参数序对齐 PDF 命令格式: [SVLAN=,]CVLAN[,UV][,DESC][,SCOS][,CCOS]。
func addPONVLAN(ctx context.Context, d CmdSink, p PONVLANParams) error {
	payload := []KV{{"CVLAN", strconv.Itoa(p.CVLAN)}, {"UV", strconv.Itoa(p.UV)}, {"DESC", p.Name}}
	if p.HasSVLAN {
		payload = append([]KV{{"SVLAN", strconv.Itoa(p.SVLAN)}}, payload...)
	}
	if p.SCOS != 0 {
		payload = append(payload, KV{"SCOS", strconv.Itoa(p.SCOS)})
	}
	if p.CCOS != 0 {
		payload = append(payload, KV{"CCOS", strconv.Itoa(p.CCOS)})
	}
	return writeCmd(ctx, d, Command{Verb: "ADD-PONVLAN", Access: accessONU(p.OLTID, p.PONID, p.ONUIDType, p.ONUID), Payload: payload})
}

func accessONU(oltid, ponid, idType, id string) []KV {
	return []KV{{"OLTID", oltid}, {"PONID", ponid}, {"ONUIDTYPE", idType}, {"ONUID", id}}
}

// query 执行查询命令:COMPLD 且 EN=0 返回行;否则归一为 *CmdError。
func query(ctx context.Context, d CmdSink, c Command) ([]map[string]string, error) {
	resp, err := d.Do(ctx, c)
	if err != nil {
		return nil, err
	}
	if err := judge(c, resp); err != nil {
		return nil, err
	}
	return resp.Rows, nil
}

// writeCmd 执行写命令:COMPLD 且 EN=0 视为成功;不重发(非幂等)。
func writeCmd(ctx context.Context, d CmdSink, c Command) error {
	resp, err := d.Do(ctx, c)
	if err != nil {
		return err
	}
	return judge(c, resp)
}

// judge 统一判定:非 COMPLD 或 EN!=0 一律 *CmdError(带 EN/ENDESC/ctag)。
func judge(c Command, resp *Response) error {
	if resp == nil {
		return fmt.Errorf("%w: %s empty response", ErrParse, c.Verb)
	}
	if resp.Completion == "COMPLD" && resp.EN == 0 {
		return nil
	}
	return &CmdError{Cmd: c.Verb, CTag: resp.CTag, EN: resp.EN, ENDESC: resp.ENDESC, Err: ErrDenied}
}

// CmdError 网元拒绝/操作失败:携带定位字段供留痕与告警日志。
type CmdError struct {
	Cmd, CTag string
	EN        int
	ENDESC    string
	Err       error
}

func (e *CmdError) Error() string {
	return fmt.Sprintf("%v: cmd=%s ctag=%s EN=%d ENDESC=%s", e.Err, e.Cmd, e.CTag, e.EN, e.ENDESC)
}

func (e *CmdError) Unwrap() error { return e.Err }
