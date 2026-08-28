package billing

import (
	"context"
	"fmt"
	"math"
)

// 柜台日结(纪要 2026-08-28-柜面现金收款,周敏口径):
// 收入/退款分列——退款按流水发生日归属当日,跨日冲销不得污染当日实点勾对;
// 净额仅汇总展示,钱箱实点分别与收入、退款两列勾对。

// cashClosingDiffTolerance 平差阈值(分):|系统净额-实点|≤0.005 元视为平账。
const cashClosingDiffTolerance = 0.005

// DailyCashSummary 按网点+操作员聚合指定日期 cash 流水,并附当日已回填实点。
// 两步查询(聚合 + 回填行)在 Go 侧合并,保持 SQL 简单可测。
func (s *PGStore) DailyCashSummary(ctx context.Context, date string) ([]DailyCashRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT COALESCE(site_name, ''), COALESCE(operator_name, ''),
		       COALESCE(SUM(CASE WHEN status = 'SUCCESS' THEN amount END), 0),
		       COALESCE(SUM(CASE WHEN status = 'REFUNDED' THEN amount END), 0)
		FROM payments
		WHERE method = 'cash' AND created_at::date = $1::date
		GROUP BY 1, 2
		ORDER BY 1, 2`, date)
	if err != nil {
		return nil, fmt.Errorf("billing: daily cash summary %s: %w", date, err)
	}
	defer rows.Close()
	counted := map[string]float64{}
	clRows, err := s.db.Query(ctx, `
		SELECT site_name, operator_name, counted_amount
		FROM payment_daily_closings WHERE closing_date = $1::date`, date)
	if err != nil {
		return nil, fmt.Errorf("billing: daily closings %s: %w", date, err)
	}
	defer clRows.Close()
	for clRows.Next() {
		var site, op string
		var amt float64
		if err := clRows.Scan(&site, &op, &amt); err != nil {
			return nil, fmt.Errorf("billing: scan closing %s: %w", date, err)
		}
		counted[site+"\x00"+op] = amt
	}
	if err := clRows.Err(); err != nil {
		return nil, fmt.Errorf("billing: iterate closings %s: %w", date, err)
	}
	out := []DailyCashRow{}
	for rows.Next() {
		var r DailyCashRow
		if err := rows.Scan(&r.SiteName, &r.OperatorName, &r.InAmount, &r.RefundAmount); err != nil {
			return nil, fmt.Errorf("billing: scan summary %s: %w", date, err)
		}
		r.NetAmount = math.Round((r.InAmount-r.RefundAmount)*100) / 100
		if v, ok := counted[r.SiteName+"\x00"+r.OperatorName]; ok {
			cv := v
			r.CountedAmount = &cv
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing: iterate summary %s: %w", date, err)
	}
	return out, nil
}

// CashPaymentsByDate 指定日期 cash 流水逐笔,时间倒序(下钻含 REFUNDED 凭证及原因/时间)。
func (s *PGStore) CashPaymentsByDate(ctx context.Context, date string) ([]Payment, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, pay_no, COALESCE(bill_id, 0), COALESCE(customer_id, 0),
		       amount, status, refund_reason, refunded_at,
		       COALESCE(site_name, ''), COALESCE(counter_code, ''), COALESCE(operator_name, '')
		FROM payments
		WHERE method = 'cash' AND created_at::date = $1::date
		ORDER BY created_at DESC, id DESC`, date)
	if err != nil {
		return nil, fmt.Errorf("billing: cash payments %s: %w", date, err)
	}
	defer rows.Close()
	out := []Payment{}
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.PayNo, &p.BillID, &p.CustomerID,
			&p.Amount, &p.Status, &p.RefundReason, &p.RefundedAt,
			&p.SiteName, &p.CounterCode, &p.OperatorName); err != nil {
			return nil, fmt.Errorf("billing: scan cash payment %s: %w", date, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing: iterate cash payments %s: %w", date, err)
	}
	return out, nil
}

// SaveDailyClosing 实点回填:UPSERT 当日快照(系统净额随回填重算落库),
// 返回差异;不平不阻塞回填,由 handler 输出 [paycheck] DIFF 日志。
func (s *PGStore) SaveDailyClosing(ctx context.Context, cl DailyClosing) (DailyClosingResult, error) {
	var sys float64
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN status = 'SUCCESS' THEN amount ELSE -amount END), 0)
		FROM payments
		WHERE method = 'cash' AND created_at::date = $1::date
		  AND COALESCE(site_name, '') = $2 AND COALESCE(operator_name, '') = $3`,
		cl.Date, cl.SiteName, cl.OperatorName).Scan(&sys)
	if err != nil {
		return DailyClosingResult{}, fmt.Errorf("billing: closing system amount %s: %w", cl.Date, err)
	}
	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO payment_daily_closings
		    (closing_date, site_name, operator_name, system_amount, counted_amount, created_by)
		VALUES ($1::date, $2, $3, $4, $5, $6)
		ON CONFLICT (closing_date, site_name, operator_name)
		DO UPDATE SET system_amount = EXCLUDED.system_amount,
		              counted_amount = EXCLUDED.counted_amount,
		              created_by = EXCLUDED.created_by
		RETURNING id`,
		cl.Date, cl.SiteName, cl.OperatorName, sys, cl.CountedAmount, cl.CreatedBy).Scan(&id)
	if err != nil {
		return DailyClosingResult{}, fmt.Errorf("billing: upsert closing %s: %w", cl.Date, err)
	}
	diff := math.Round((sys-cl.CountedAmount)*100) / 100
	return DailyClosingResult{
		ID:           id,
		SystemAmount: sys,
		DiffAmount:   diff,
		Balanced:     math.Abs(diff) <= cashClosingDiffTolerance,
	}, nil
}
