package resource

// 稽核 SQL 清单与执行(P5-W3)。单检查一条只读 SQL,统一返回
// (全量计数 count(*) OVER(), id, code, detail) 形状;明细 LIMIT 采样,
// 计数不受影响。检查码登记 docs/contract/fields.md §4.2。

import (
	"context"
	"fmt"
	"time"
)

// auditCheck 单项稽核定义。
type auditCheck struct {
	code     string // 检查码(API/快照/fields.md 对齐键)
	category string // ownership/state/coding
	title    string // 中文口径(快照/日志可读)
	sql      string // 返回 (total, id, code, detail);$1 仅 staleArg 检查使用
	staleArg bool   // true=SQL 吃 $1=RESERVED 阈值小时数
}

// auditChecks 三类六查(任务书范围,不附加):归属断裂二查/状态机违例二查/编码规范违例二查。
var auditChecks = []auditCheck{
	{
		code: "PORT_SPLITTER_MISSING", category: AuditCatOwnership, title: "端口引用的分光器不存在",
		sql: `SELECT count(*) OVER(), p.id, p.port_code, 'resourceId=' || p.resource_id FROM ports p LEFT JOIN resources r ON r.id = p.resource_id WHERE r.id IS NULL`,
	},
	{
		code: "SPLITTER_UPSTREAM_MISSING", category: AuditCatOwnership, title: "分光器引用的上游不存在",
		sql: `SELECT count(*) OVER(), r.id, r.code, COALESCE('parentId=' || r.parent_id, 'parentId IS NULL') FROM resources r LEFT JOIN resources up ON up.id = r.parent_id WHERE r.type = 'SPLITTER' AND (r.parent_id IS NULL OR up.id IS NULL)`,
	},
	{
		code: "USED_PORT_NO_QUAD_LINK", category: AuditCatState, title: "USED 态端口无四码 LINKED 关联",
		sql: `SELECT count(*) OVER(), p.id, p.port_code, 'orderId=' || COALESCE(p.order_id, 0)::text FROM ports p WHERE p.status = 'USED' AND NOT EXISTS (SELECT 1 FROM quad_links q WHERE q.port_id = p.id AND q.status = 'LINKED')`,
	},
	{
		code: "RESERVED_PORT_STALE", category: AuditCatState, title: "RESERVED 态超阈值未推进未释放", staleArg: true,
		sql: `SELECT count(*) OVER(), p.id, p.port_code, 'since=' || to_char(COALESCE((SELECT max(h.changed_at) FROM port_change_history h WHERE h.port_id = p.id), p.created_at), 'YYYY-MM-DD HH24:MI') FROM ports p WHERE p.status = 'RESERVED' AND COALESCE((SELECT max(h.changed_at) FROM port_change_history h WHERE h.port_id = p.id), p.created_at) < now() - make_interval(hours => $1)`,
	},
	{
		code: "RESOURCE_CODE_BAD", category: AuditCatCoding, title: "资源编码不合规(须 OLT-*/SPL-* 含中缀横杠)",
		sql: `SELECT count(*) OVER(), r.id, r.code, 'type=' || r.type FROM resources r WHERE NOT (r.type = 'OLT' AND r.code ~ '^OLT-[A-Za-z0-9-]+$') AND NOT (r.type = 'SPLITTER' AND r.code ~ '^SPL-[A-Za-z0-9-]+$')`,
	},
	{
		code: "PORT_CODE_BAD", category: AuditCatCoding, title: "端口编码不合规(须 P-设备码-序号,紧凑/全码两形态)",
		sql: `SELECT count(*) OVER(), p.id, p.port_code, 'resource=' || r.code FROM ports p JOIN resources r ON r.id = p.resource_id WHERE (r.code ~ '^OLT-[A-Za-z0-9-]+$' OR r.code ~ '^SPL-[A-Za-z0-9-]+$') AND p.port_code !~ ('^P-' || replace(r.code, '-', '') || '-[A-Za-z0-9]+$') AND p.port_code !~ ('^P-' || r.code || '-[A-Za-z0-9]+$')`,
	},
}

// AuditInventory 执行稽核(只读),按 opts.Category 过滤类别(空=全部三类)。
// 单检查失败即整轮报错(只读失败=连接问题,与巡检同语义)。
func (s *PGStore) AuditInventory(ctx context.Context, opts AuditOptions) (*AuditReport, error) {
	stale := opts.ReservedStaleHours
	if stale <= 0 {
		stale = AuditDefaultStaleHours
	}
	rep := &AuditReport{
		GeneratedAt: time.Now(), StaleHours: stale,
		Counts: []AuditCategoryCount{}, Items: []AuditItem{},
	}
	for _, ck := range auditChecks {
		if opts.Category != "" && ck.category != opts.Category {
			continue
		}
		total, items, err := s.runAuditCheck(ctx, ck, stale)
		if err != nil {
			return nil, err
		}
		rep.addCheck(ck.category, total)
		rep.Items = append(rep.Items, items...)
	}
	return rep, nil
}

// addCheck 累加类别计数(类别首次出现即建行;Total 同步累加)。
func (r *AuditReport) addCheck(category string, total int64) {
	r.Total += total
	for i := range r.Counts {
		if r.Counts[i].Category == category {
			r.Counts[i].Count += total
			return
		}
	}
	r.Counts = append(r.Counts, AuditCategoryCount{Category: category, Count: total})
}

// runAuditCheck 跑单项检查,返回全量计数与明细样本。
func (s *PGStore) runAuditCheck(ctx context.Context, ck auditCheck, staleHours int) (int64, []AuditItem, error) {
	q := ck.sql + " LIMIT " + fmt.Sprint(AuditSampleLimit)
	var args []any
	if ck.staleArg {
		args = append(args, staleHours)
	}
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return 0, nil, fmt.Errorf("resource: audit %s: %w", ck.code, err)
	}
	defer rows.Close()
	var total int64
	items := []AuditItem{}
	for rows.Next() {
		var it AuditItem
		it.Check = ck.code
		it.Category = ck.category
		if err := rows.Scan(&total, &it.ID, &it.Code, &it.Detail); err != nil {
			return 0, nil, fmt.Errorf("resource: audit %s scan: %w", ck.code, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("resource: audit %s rows: %w", ck.code, err)
	}
	return total, items, nil
}
