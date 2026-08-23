package order

import (
	"context"
	"fmt"
)

func (s *PGStore) CSMetrics(ctx context.Context) (*CSMetrics, error) {
	var m CSMetrics
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'OPEN'), COUNT(*) FILTER (WHERE status = 'PROCESSING'), COUNT(*) FILTER (WHERE status = 'CLOSED'),
		COUNT(*) FILTER (WHERE status <> 'CLOSED' AND sla_deadline IS NOT NULL AND sla_deadline < now()),
		COUNT(*) FILTER (WHERE status = 'CLOSED' AND closed_at IS NOT NULL AND sla_deadline IS NOT NULL AND closed_at <= sla_deadline),
		COUNT(*) FILTER (WHERE status = 'CLOSED' AND closed_at IS NOT NULL AND sla_deadline IS NOT NULL AND closed_at > sla_deadline),
		COALESCE(AVG(EXTRACT(EPOCH FROM (closed_at - created_at)) / 3600) FILTER (WHERE status = 'CLOSED' AND closed_at IS NOT NULL), 0)
		FROM complaints`).Scan(&m.OpenCount, &m.ProcessingCount, &m.ClosedCount, &m.SLABreachedOpen, &m.SLAOnTimeClosed, &m.SLAOverdueClosed, &m.AvgCloseHours)
	if err != nil {
		return nil, fmt.Errorf("order: cs metrics: %w", err)
	}
	return &m, nil
}
