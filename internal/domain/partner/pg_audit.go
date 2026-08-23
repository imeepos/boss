package partner

import (
	"context"
	"fmt"
	"time"
)

func (s *PGStore) ListAuditReport(ctx context.Context, accountID int64, action string) ([]AuditReportRow, error) {
	entityID, err := s.entityOfAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if entityID == 0 {
		return nil, ErrNotPartner
	}
	rows, err := s.db.Query(ctx, `
SELECT l.action, count(*)::bigint, max(l.created_at)
FROM audit_logs l
JOIN orders o ON o.order_no = l.target_id AND l.target_type IN ('order','partner_order')
WHERE o.legal_entity_id=$1 AND ($2='' OR l.action=$2)
GROUP BY l.action ORDER BY count(*) DESC, l.action`, entityID, action)
	if err != nil {
		return nil, fmt.Errorf("partner: audit report: %w", err)
	}
	defer rows.Close()
	out := make([]AuditReportRow, 0)
	for rows.Next() {
		var item AuditReportRow
		var last *time.Time
		if err := rows.Scan(&item.Action, &item.Count, &last); err != nil {
			return nil, err
		}
		if last != nil {
			item.LastAt = last.Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
