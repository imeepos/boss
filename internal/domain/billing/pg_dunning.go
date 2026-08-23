package billing

import (
	"context"
	"fmt"
	"math"
	"time"
)

// MarkOverdueBills 宽限外未缴账单置 OVERDUE;账龄自 created_at 起算(无独立到期日列)。
func (s *PGStore) MarkOverdueBills(ctx context.Context, graceDays int) (int, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE bills SET status = 'OVERDUE'
		WHERE status = 'UNPAID' AND created_at < now() - make_interval(days => $1)`, graceDays)
	if err != nil {
		return 0, fmt.Errorf("billing: mark overdue: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ListOverdueCustomers 逾期客户清单(按客户聚合,最早逾期在前)。
func (s *PGStore) ListOverdueCustomers(ctx context.Context) ([]OverdueCustomer, error) {
	rows, err := s.db.Query(ctx, `
		SELECT customer_id, SUM(amount), MIN(created_at)
		FROM bills WHERE status = 'OVERDUE'
		GROUP BY customer_id ORDER BY MIN(created_at)`)
	if err != nil {
		return nil, fmt.Errorf("billing: list overdue customers: %w", err)
	}
	defer rows.Close()
	out := make([]OverdueCustomer, 0)
	for rows.Next() {
		var it OverdueCustomer
		if err := rows.Scan(&it.CustomerID, &it.Amount, &it.OldestAt); err != nil {
			return nil, fmt.Errorf("billing: scan overdue customer: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ArrearsDaysFrom 欠费天数 = 今天 - 最早逾期日(不足一天按 0 天,不夸大)。
func ArrearsDaysFrom(now, oldest time.Time) int32 {
	d := int32(math.Floor(now.Sub(oldest).Hours() / 24))
	if d < 0 {
		return 0
	}
	return d
}
