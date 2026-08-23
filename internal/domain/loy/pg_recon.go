package loy

import (
	"context"
	"fmt"
)

// 积分对账差异种类。
const (
	ReconDiffMatch        = "MATCH"         // 账本余额 = 流水合计
	ReconDiffBalanceDrift = "BALANCE_DRIFT" // 账本余额 ≠ 流水合计
)

// PointReconRow 积分对账行:一客户的账本余额 vs 流水合计三角。
type PointReconRow struct {
	CustomerID   int64            `json:"customerId"`
	Balance      int64            `json:"balance"`      // 账本余额(唯一事实源口径)
	EntriesSum   int64            `json:"entriesSum"`   // 流水 delta 合计
	LifetimeEarn int64            `json:"lifetimeEarn"` // 累计获得(正向合计)
	ExpiredTotal int64            `json:"expiredTotal"` // 已过期清算合计(负向绝对值)
	EntryCount   int64            `json:"entryCount"`
	ByReason     map[string]int64 `json:"byReason"`
	DiffKind     string           `json:"diffKind"`
}

// PointReconSummary 积分对账汇总。
type PointReconSummary struct {
	Customers    int            `json:"customers"`
	BalanceTotal int64          `json:"balanceTotal"`
	LifetimeEarn int64          `json:"lifetimeEarn"`
	ExpiredTotal int64          `json:"expiredTotal"`
	ByDiff       map[string]int `json:"byDiff"`
}

// PointsRecon 积分对账报表(全账本,仅列 drift 或前 N 全量由调用方筛)。
func (s *PGStore) PointsRecon(ctx context.Context) ([]PointReconRow, PointReconSummary, error) {
	rows, err := s.db.Query(ctx, `
		SELECT l.customer_id, l.balance,
		       COALESCE(SUM(e.delta),0),
		       COALESCE(SUM(e.delta) FILTER (WHERE e.delta>0),0),
		       COALESCE(-SUM(e.delta) FILTER (WHERE e.reason='EXPIRED'),0),
		       COUNT(e.entry_id)
		FROM loy_point_ledgers l
		LEFT JOIN loy_point_entries e ON e.customer_id = l.customer_id
		GROUP BY l.customer_id, l.balance
		ORDER BY l.customer_id`)
	if err != nil {
		return nil, PointReconSummary{}, fmt.Errorf("loy: points recon query: %w", err)
	}
	defer rows.Close()
	out := make([]PointReconRow, 0)
	for rows.Next() {
		var r PointReconRow
		if err := rows.Scan(&r.CustomerID, &r.Balance, &r.EntriesSum,
			&r.LifetimeEarn, &r.ExpiredTotal, &r.EntryCount); err != nil {
			return nil, PointReconSummary{}, fmt.Errorf("loy: points recon scan: %w", err)
		}
		r.DiffKind = classifyPointRecon(r.Balance, r.EntriesSum)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, PointReconSummary{}, fmt.Errorf("loy: points recon rows: %w", err)
	}
	if err := s.fillReasons(ctx, out); err != nil {
		return nil, PointReconSummary{}, err
	}

	sum := PointReconSummary{Customers: len(out), ByDiff: map[string]int{}}
	for _, r := range out {
		sum.BalanceTotal += r.Balance
		sum.LifetimeEarn += r.LifetimeEarn
		sum.ExpiredTotal += r.ExpiredTotal
		sum.ByDiff[r.DiffKind]++
	}
	return out, sum, nil
}

// fillReasons 流水原因分布(每客户 reason→合计 delta;ANY 数组单轮取全)。
func (s *PGStore) fillReasons(ctx context.Context, rows []PointReconRow) error {
	if len(rows) == 0 {
		return nil
	}
	rrows, err := s.db.Query(ctx, `
		SELECT customer_id, reason, SUM(delta)
		FROM loy_point_entries WHERE customer_id = ANY($1)
		GROUP BY customer_id, reason`, idsOf(rows))
	if err != nil {
		return fmt.Errorf("loy: reasons query: %w", err)
	}
	defer rrows.Close()
	byCust := map[int64]map[string]int64{}
	for rrows.Next() {
		var cid int64
		var reason string
		var delta int64
		if err := rrows.Scan(&cid, &reason, &delta); err != nil {
			return fmt.Errorf("loy: reasons scan: %w", err)
		}
		if byCust[cid] == nil {
			byCust[cid] = map[string]int64{}
		}
		byCust[cid][reason] = delta
	}
	if err := rrows.Err(); err != nil {
		return fmt.Errorf("loy: reasons rows: %w", err)
	}
	for i := range rows {
		rows[i].ByReason = byCust[rows[i].CustomerID]
	}
	return nil
}

func idsOf(rows []PointReconRow) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.CustomerID)
	}
	return ids
}

// classifyPointRecon 差异判定:账本余额与流水合计一致即 MATCH,否则漂移。
func classifyPointRecon(balance, entriesSum int64) string {
	if balance == entriesSum {
		return ReconDiffMatch
	}
	return ReconDiffBalanceDrift
}
