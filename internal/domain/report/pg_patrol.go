package report

// 软引用孤儿巡检(db-design-review D7):跨域软引用无 FK 约束,删主档可留孤儿。
// 只读、只报表不阻断;admin GET /db-patrol/orphans(menu:report)。

import (
	"context"
	"errors"
	"fmt"
)

// ErrNoPatrol Store 实现不支持巡检(如测试 fake)。
var ErrNoPatrol = errors.New("report: store does not support orphan patrol")

// OrphanPatroller Store 可选能力:软引用孤儿巡检。
type OrphanPatroller interface {
	PatrolOrphans(ctx context.Context) ([]OrphanFinding, error)
}

// PatrolOrphans ReportService 入口;St 不支持时返回 ErrNoPatrol。
func (r *ReportService) PatrolOrphans(ctx context.Context) ([]OrphanFinding, error) {
	p, ok := r.St.(OrphanPatroller)
	if !ok {
		return nil, ErrNoPatrol
	}
	return p.PatrolOrphans(ctx)
}

// OrphanFinding 一条巡检结果:某软引用关系下的孤儿行数 + 样本主键。
type OrphanFinding struct {
	Check     string  `json:"check"`     // 检查名:子表.列 -> 父表
	Orphans   int     `json:"orphans"`   // 孤儿行数
	SampleIDs []int64 `json:"sampleIds"` // 样本(最多10个)
}

// orphanChecks 巡检清单(软引用关系,字段权威 data-relations.md §0.5)。
// 每项 SQL 返回 (count, array_agg(sample id))。
var orphanChecks = []struct {
	name string
	sql  string
}{
	{"orders.customer_id -> customers", `SELECT count(*), COALESCE((array_agg(o.id ORDER BY o.id))[1:10], '{}'::bigint[])
		FROM orders o WHERE o.customer_id > 0 AND NOT EXISTS (SELECT 1 FROM customers c WHERE c.id = o.customer_id)`},
	{"orders.offer_id -> product_offers", `SELECT count(*), COALESCE((array_agg(o.id ORDER BY o.id))[1:10], '{}'::bigint[])
		FROM orders o WHERE o.offer_id > 0 AND NOT EXISTS (SELECT 1 FROM product_offers p WHERE p.id = o.offer_id)`},
	{"orders.channel_id -> channels", `SELECT count(*), COALESCE((array_agg(o.id ORDER BY o.id))[1:10], '{}'::bigint[])
		FROM orders o WHERE o.channel_id > 0 AND NOT EXISTS (SELECT 1 FROM channels ch WHERE ch.id = o.channel_id)`},
	{"lo_accounts.customer_id -> customers", `SELECT count(*), COALESCE((array_agg(l.id ORDER BY l.id))[1:10], '{}'::bigint[])
		FROM lo_accounts l WHERE l.customer_id > 0 AND NOT EXISTS (SELECT 1 FROM customers c WHERE c.id = l.customer_id)`},
	{"lo_accounts.qos_template_id -> qos_templates", `SELECT count(*), COALESCE((array_agg(l.id ORDER BY l.id))[1:10], '{}'::bigint[])
		FROM lo_accounts l WHERE l.qos_template_id > 0 AND NOT EXISTS (SELECT 1 FROM qos_templates q WHERE q.id = l.qos_template_id)`},
	{"reserve_records.port_id -> ports", `SELECT count(*), COALESCE((array_agg(r.id ORDER BY r.id))[1:10], '{}'::bigint[])
		FROM reserve_records r WHERE r.port_id > 0 AND NOT EXISTS (SELECT 1 FROM ports p WHERE p.id = r.port_id)`},
	{"reserve_records.order_id -> orders", `SELECT count(*), COALESCE((array_agg(r.id ORDER BY r.id))[1:10], '{}'::bigint[])
		FROM reserve_records r WHERE r.order_id > 0 AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.id = r.order_id)`},
	{"transfers.resource_id -> resources", `SELECT count(*), COALESCE((array_agg(t.id ORDER BY t.id))[1:10], '{}'::bigint[])
		FROM transfers t WHERE t.resource_id > 0 AND NOT EXISTS (SELECT 1 FROM resources rs WHERE rs.id = t.resource_id)`},
	{"alarms.resource_id -> resources", `SELECT count(*), COALESCE((array_agg(a.id ORDER BY a.id))[1:10], '{}'::bigint[])
		FROM alarms a WHERE a.resource_id > 0
		AND NOT EXISTS (SELECT 1 FROM resources rs WHERE rs.id = a.resource_id)`},
	{"invoices.customer_id -> customers", `SELECT count(*), COALESCE((array_agg(i.id ORDER BY i.id))[1:10], '{}'::bigint[])
		FROM invoices i WHERE i.customer_id > 0 AND NOT EXISTS (SELECT 1 FROM customers c WHERE c.id = i.customer_id)`},
	{"provision_logs.task_id -> provision_tasks", `SELECT count(*), COALESCE((array_agg(l.id ORDER BY l.id))[1:10], '{}'::bigint[])
		FROM provision_logs l WHERE l.task_id > 0 AND NOT EXISTS (SELECT 1 FROM provision_tasks t WHERE t.id = l.task_id)`},
}

// PatrolOrphans 逐项跑巡检;单项 SQL 失败即中止(巡检只读,失败=连接问题)。
func (s *PGStore) PatrolOrphans(ctx context.Context) ([]OrphanFinding, error) {
	out := make([]OrphanFinding, 0, len(orphanChecks))
	for _, ck := range orphanChecks {
		var n int
		var ids []int64
		if err := s.db.QueryRow(ctx, ck.sql).Scan(&n, &ids); err != nil {
			return nil, fmt.Errorf("report: patrol %s: %w", ck.name, err)
		}
		out = append(out, OrphanFinding{Check: ck.name, Orphans: n, SampleIDs: ids})
	}
	return out, nil
}
