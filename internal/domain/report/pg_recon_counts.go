package report

// 每日对账只读计数(Q2/S2):0=OK,非 0=异常条数,由 DailyRecon 落快照。
// S2 扩展:发票、积分、GIS 投影、开放平台投递。
// 全部为单条 count SQL,失败即连接问题,整轮报错。

import (
	"context"
	"fmt"
)

// reconQuery 单域检查定义。
type reconQuery struct {
	domain string
	name   string
	detail string
	sql    string
}

// reconQueries 检查清单(订单×2/四码/资源/账务/GIS/发票/积分/Webhook 口径)。
var reconQueries = []reconQuery{
	{
		domain: "order", name: "stuckReserved", detail: "RESERVED 超 24h 未推进(超时释放循环应清零)",
		sql: `SELECT count(*) FROM orders o WHERE o.status = 'RESERVED'
		      AND COALESCE((SELECT max(st.finished_at) FROM order_stages st
		                    WHERE st.order_id = o.id AND st.stage = 3), o.created_at) < now() - interval '24 hours'`,
	},
	{
		domain: "order", name: "doneStageMismatch", detail: "status=DONE 但环节未到 12",
		sql: `SELECT count(*) FROM orders WHERE status = 'DONE' AND stage <> 12`,
	},
	{
		domain: "quadlink", name: "conflicts", detail: "四码冲突未清零(时限 4 小时,terms.md)",
		sql: `SELECT count(*) FROM quad_links WHERE status = 'CONFLICT'`,
	},
	{
		domain: "resource", name: "orphanReservedPorts", detail: "端口 RESERVED 但无在途 RESERVED 订单",
		sql: `SELECT count(*) FROM ports p WHERE p.status = 'RESERVED'
		      AND NOT EXISTS (SELECT 1 FROM orders o WHERE o.id = p.order_id AND o.status = 'RESERVED')`,
	},
	{
		domain: "billing", name: "diffPendingOld", detail: "缴费对账批次差异挂起超 48h 未平账",
		sql: `SELECT count(*) FROM reconciliation_batches
		      WHERE status = 'DIFF_PENDING' AND created_at < now() - interval '48 hours'`,
	},
	{
		domain: "gis", name: "usedPortNoOrder", detail: "地图口径:端口 USED 但订单引用悬空",
		sql: `SELECT count(*) FROM ports p WHERE p.status = 'USED'
		      AND (p.order_id IS NULL OR NOT EXISTS (SELECT 1 FROM orders o WHERE o.id = p.order_id AND o.status IN ('INSTALLING','DONE')))`,
	},
	{
		domain: "billing", name: "taxFailed", detail: "税局开具失败待重试",
		sql: `SELECT count(*) FROM invoices WHERE tax_status = 'FAILED'`,
	},
	{
		domain: "loy", name: "pointsReconDiff", detail: "积分流水与账本差异非零",
		sql: `SELECT count(*) FROM (
			SELECT lt.id FROM loy_ledgers lt
			LEFT JOIN loy_entries le ON le.ledger_id = lt.id
			WHERE lt.balance != COALESCE((SELECT SUM(le.amount) FROM loy_entries le WHERE le.ledger_id = lt.id), 0)
			LIMIT 100
		) sub`,
	},
	{
		domain: "gis", name: "projectionStale", detail: "GIS 投影超过 24h 未更新",
		sql: `SELECT count(*) FROM gis_points WHERE updated_at < now() - interval '24 hours'`,
	},
	{
		domain: "openplat", name: "webhookFailures", detail: "Webhook 投递失败超 3 次",
		sql: `SELECT count(*) FROM open_webhook_deliveries WHERE status = 'FAILED' AND retry_count >= 3`,
	},
}

// ReconCounts 逐域执行计数检查,返回与 reconQueries 同序的结果。
func (s *PGStore) ReconCounts(ctx context.Context) ([]ReconCheck, error) {
	out := make([]ReconCheck, 0, len(reconQueries))
	for _, q := range reconQueries {
		var n int64
		if err := s.db.QueryRow(ctx, q.sql).Scan(&n); err != nil {
			return nil, fmt.Errorf("report: recon %s.%s: %w", q.domain, q.name, err)
		}
		out = append(out, ReconCheck{Domain: q.domain, Name: q.name, Detail: q.detail, Count: n})
	}
	return out, nil
}
