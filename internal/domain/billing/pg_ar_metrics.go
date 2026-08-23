package billing

import (
	"context"
	"fmt"
)

func (s *PGStore) ARMetrics(ctx context.Context) (*ARMetrics, error) {
	var m ARMetrics
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE((SELECT SUM(amount) FROM arrears), 0), (SELECT COUNT(*) FROM arrears),
		(SELECT COUNT(*) FROM arrears WHERE status = 'STOPPED'), (SELECT COUNT(*) FROM bills WHERE status = 'OVERDUE'),
		(SELECT COUNT(*) FILTER (WHERE days BETWEEN 0 AND 15) FROM arrears), (SELECT COUNT(*) FILTER (WHERE days BETWEEN 16 AND 30) FROM arrears),
		(SELECT COUNT(*) FILTER (WHERE days BETWEEN 31 AND 60) FROM arrears), (SELECT COUNT(*) FILTER (WHERE days BETWEEN 61 AND 90) FROM arrears),
		(SELECT COUNT(*) FILTER (WHERE days > 90) FROM arrears), COALESCE((SELECT MAX(updated_at)::text FROM arrears), '')`).Scan(
		&m.TotalAmount, &m.CustomerCount, &m.StoppedCount, &m.OverdueBillCount, &m.AgingBuckets.D0To15, &m.AgingBuckets.D16To30,
		&m.AgingBuckets.D31To60, &m.AgingBuckets.D61To90, &m.AgingBuckets.D90Plus, &m.LastRunAt)
	if err != nil {
		return nil, fmt.Errorf("billing: ar metrics: %w", err)
	}
	return &m, nil
}
