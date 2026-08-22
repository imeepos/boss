package billing

import (
	"context"
	"fmt"
)

// AppendTaxEvent 追加轨迹行;调用方在状态迁移成功后落痕(设计取舍见 docs/design/q3-tax-trail.md)。
func (s *PGStore) AppendTaxEvent(ctx context.Context, e TaxEvent) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO invoice_tax_events(invoice_id, event, tax_status_after, tax_no, fail_reason, operator_account_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		e.InvoiceID, e.Event, e.TaxStatusAfter, e.TaxNo, e.FailReason, e.OperatorAccountID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: append tax event: %w", err)
	}
	return id, nil
}

// ListTaxEvents 按发票回放轨迹(时间正序)。
func (s *PGStore) ListTaxEvents(ctx context.Context, invoiceID int64) ([]TaxEvent, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, invoice_id, event, tax_status_after, tax_no, fail_reason, operator_account_id, created_at
		FROM invoice_tax_events WHERE invoice_id = $1 ORDER BY created_at, id`, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("billing: list tax events: %w", err)
	}
	defer rows.Close()
	out := make([]TaxEvent, 0)
	for rows.Next() {
		var e TaxEvent
		if err := rows.Scan(&e.ID, &e.InvoiceID, &e.Event, &e.TaxStatusAfter, &e.TaxNo,
			&e.FailReason, &e.OperatorAccountID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("billing: scan tax event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
