// TL1 下发参数解析器(设计 §4):从 provision 域取数链一次取齐下发全部参数。
// 缺任一项返回带明确原因的 error(任务 FAILED 可诊断);不做任何网元通信(那是 executor 的事)。
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// 缺数据四类 FAIL 文案(逐字,任务 FAILED 留痕用)。
const (
	errLOAccountMissing = "lo account missing"
	errPortPONMissing   = "port missing PON positioning"
	errOLTMissingNMSID  = "OLT missing nms_oltid"
)

// dbTL1 取数链最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbTL1 interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// TL1ParamResolver 实现 tl1.ParamResolver;端点/参数取数优先表行,env 兜底。
type TL1ParamResolver struct {
	db dbTL1
}

// NewTL1ParamResolver 构造解析器;pool 供取数链与 ONUNO 分配。
func NewTL1ParamResolver(pool *pgxpool.Pool) *TL1ParamResolver {
	return &TL1ParamResolver{db: pool}
}

// templateContent JSONB content 形态:{"onuType":..,"services":{<名>:<流配置>}}。
// services 键名即服务名(DESC=PRV-<orderNo>-<键名>);tr069 无 svlan 键则 HasSVLAN=false。
type templateContent struct {
	ONUType  string                  `json:"onuType"`
	Services map[string]tl1SvcConfig `json:"services"`
}

// tl1SvcConfig 单业务流参数;SVLAN 指针区分"无 svlan 键"与"显式 0"。
type tl1SvcConfig struct {
	SVLAN *int `json:"svlan"`
	CVLAN int  `json:"cvlan"`
	UV    int  `json:"uv"`
	SCOS  int  `json:"scos"`
	CCOS  int  `json:"ccos"`
}

// Resolve 一次取齐下发参数;任一步缺数据/查库失败返回带原因的 error。
func (r *TL1ParamResolver) Resolve(ctx context.Context, t provision.Task) (tl1.Params, error) {
	p := tl1.Params{AuthType: "LOID"}

	orderNo, customerID, legalEntityID, err := r.orderOf(ctx, t.OrderID)
	if err != nil {
		return p, r.fail(t, err)
	}
	p.Desc = "PRV-" + orderNo

	p.ONUID, err = r.loidOf(ctx, customerID)
	if err != nil {
		return p, r.fail(t, err)
	}

	onuType, svcs, err := r.contentOf(ctx, t.TemplateID)
	if err != nil {
		return p, r.fail(t, err)
	}
	p.ONUType = onuType

	port, err := r.portOf(ctx, t.OrderID)
	if err != nil {
		return p, r.fail(t, err)
	}
	p.PONID = fmt.Sprintf("NA-%d-%d-%d", port.ponFrame, port.ponSlot, port.ponPort)

	p.OLTID, err = r.oltidOf(ctx, port.resourceID)
	if err != nil {
		return p, r.fail(t, err)
	}

	p.Endpoint, err = r.endpointOf(ctx, legalEntityID)
	if err != nil {
		return p, r.fail(t, err)
	}

	p.ONUNo, err = r.onuNo(ctx, port)
	if err != nil {
		return p, r.fail(t, err)
	}

	p.Services = buildServices(p, svcs)
	return p, nil
}

// orderOf 订单行:order_no/customer_id/legal_entity_id。
func (r *TL1ParamResolver) orderOf(ctx context.Context, orderID int64) (string, int64, int64, error) {
	var orderNo string
	var customerID, legalEntityID int64
	err := r.db.QueryRow(ctx, `
		SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = $1`, orderID).
		Scan(&orderNo, &customerID, &legalEntityID)
	if err != nil {
		return "", 0, 0, fmt.Errorf("order %d: %w", orderID, err)
	}
	return orderNo, customerID, legalEntityID, nil
}

// loidOf LO 账号 loid(环节6 已建);缺=FAIL "lo account missing"。
func (r *TL1ParamResolver) loidOf(ctx context.Context, customerID int64) (string, error) {
	var loid string
	err := r.db.QueryRow(ctx, `
		SELECT loid FROM lo_accounts WHERE customer_id = $1 AND status = 'ACTIVE' ORDER BY id LIMIT 1`, customerID).
		Scan(&loid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New(errLOAccountMissing)
	}
	if err != nil {
		return "", fmt.Errorf("lo_accounts: %w", err)
	}
	return loid, nil
}

// contentOf 模板 content JSONB → ONUType + 服务配置。
func (r *TL1ParamResolver) contentOf(ctx context.Context, templateID int64) (string, map[string]tl1SvcConfig, error) {
	var raw []byte
	err := r.db.QueryRow(ctx, `SELECT content FROM provision_templates WHERE id = $1`, templateID).Scan(&raw)
	if err != nil {
		return "", nil, fmt.Errorf("template %d: %w", templateID, err)
	}
	var c templateContent
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", nil, fmt.Errorf("template %d content: %w", templateID, err)
	}
	return c.ONUType, c.Services, nil
}

// portRow 端口定位行(resource_id + PON 三维 + onu_no)。
type portRow struct {
	id         int64
	resourceID int64
	ponFrame   int
	ponSlot    int
	ponPort    int
	onuNo      sql.NullInt16
}

// portOf 订单占用端口(恰 1 行,RESERVED/USED);缺行/缺 PON 定位 FAIL。
func (r *TL1ParamResolver) portOf(ctx context.Context, orderID int64) (portRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no
		FROM ports WHERE order_id = $1 AND status IN ('RESERVED', 'USED')`, orderID)
	if err != nil {
		return portRow{}, fmt.Errorf("ports: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return portRow{}, errors.New(errPortPONMissing)
	}
	var p portRow
	var frame, slot, port sql.NullInt16
	if err := rows.Scan(&p.id, &p.resourceID, &frame, &slot, &port, &p.onuNo); err != nil {
		return portRow{}, fmt.Errorf("ports scan: %w", err)
	}
	if !frame.Valid || !slot.Valid || !port.Valid {
		return portRow{}, errors.New(errPortPONMissing)
	}
	p.ponFrame, p.ponSlot, p.ponPort = int(frame.Int16), int(slot.Int16), int(port.Int16)
	return p, nil
}

// oltidOf OLT 资源行的 U2000 侧标识;缺=FAIL "OLT missing nms_oltid"。
func (r *TL1ParamResolver) oltidOf(ctx context.Context, resourceID int64) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT nms_oltid FROM resources WHERE id = $1 AND type = 'OLT'`, resourceID).Scan(&id)
	if err != nil || id == "" {
		return "", errors.New(errOLTMissingNMSID)
	}
	return id, nil
}

// endpointOf 端点:表行优先(解密 pass_cipher),否则 env 兜底,都缺 FAIL。
func (r *TL1ParamResolver) endpointOf(ctx context.Context, legalEntityID int64) (tl1.Endpoint, error) {
	var ep tl1.Endpoint
	var passCipher string
	err := r.db.QueryRow(ctx, `
		SELECT host, port, username, pass_cipher FROM provision_nms WHERE legal_entity_id = $1`, legalEntityID).
		Scan(&ep.Host, &ep.Port, &ep.User, &passCipher)
	if err == nil {
		ep.Pass = decryptConfigSecret("provision_nms.pass", passCipher)
		return ep, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ep, fmt.Errorf("provision_nms: %w", err)
	}
	addr := strings.TrimSpace(os.Getenv("BOSS_PROVISION_TL1_ADDR"))
	if addr == "" {
		return ep, fmt.Errorf("no NMS endpoint for entity %d", legalEntityID)
	}
	ep.Host, ep.Port = splitHostPort(addr)
	ep.User = os.Getenv("BOSS_PROVISION_TL1_USER")
	ep.Pass = os.Getenv("BOSS_PROVISION_TL1_PASS")
	return ep, nil
}

// splitHostPort 拆 host[:port],无端口默认 13027(TL1 标准端口)。
func splitHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 13027
	}
	port := 13027
	if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
		port = p
	}
	return host, port
}

// buildServices 服务配置 → tl1.PONVLANParams 列表(PONID/AuthType 已定)。
func buildServices(p tl1.Params, svcs map[string]tl1SvcConfig) []tl1.PONVLANParams {
	out := make([]tl1.PONVLANParams, 0, len(svcs))
	for name, s := range svcs {
		sv := tl1.PONVLANParams{
			ServiceName: name, OLTID: p.OLTID, PONID: p.PONID,
			ONUIDType: p.AuthType, ONUID: p.ONUID,
			CVLAN: s.CVLAN, UV: s.UV, SCOS: s.SCOS, CCOS: s.CCOS,
		}
		if s.SVLAN != nil {
			sv.SVLAN = *s.SVLAN
			sv.HasSVLAN = true
		}
		out = append(out, sv)
	}
	return out
}

// fail 失败留痕,可 grep。
func (r *TL1ParamResolver) fail(t provision.Task, err error) error {
	log.Printf("[app-tl1] RESOLVE FAILED task=%s reason=%v", t.TaskNo, err)
	return err
}
