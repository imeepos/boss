package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetInvoice 按主键查发票;未命中返回 ErrNotFound。
func (s *PGStore) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	inv, err := scanInvoice(s.db.QueryRow(ctx,
		`SELECT `+invoiceCols+` FROM invoices WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: get invoice: %w", err)
	}
	return inv, nil
}

// BackfillTaxNo 人工通道回填:税局平台开具后登记票号,税务状态置 ISSUED。
// 仅待开/失败/已提交/外部阻塞可回填(已开具不可重复回填)。
func (s *PGStore) BackfillTaxNo(ctx context.Context, id int64, taxNo string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE invoices SET tax_status = $2, tax_no = $3, tax_fail_reason = ''
		WHERE id = $1 AND tax_status IN ('PENDING','SUBMITTED','FAILED','BLOCKED')`,
		id, TaxStatusIssued, taxNo)
	if err != nil {
		return fmt.Errorf("billing: backfill tax no: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvoiceNotTaxable
	}
	return nil
}

// MarkTaxResult 落税局回执。外部事件先入轨迹，重复回执由唯一键吞掉；旧状态不被乱序回执倒退。
func (s *PGStore) MarkTaxResult(ctx context.Context, id int64, r TaxReceipt) error {
	if r.Status != TaxStatusIssued && r.Status != TaxStatusFailed && r.Status != TaxStatusSUBMITTED && r.Status != TaxStatusBlocked {
		return fmt.Errorf("billing: bad receipt status %q", r.Status)
	}
	var taxNo, failReason string
	if r.Status == TaxStatusIssued {
		taxNo = r.TaxNo
	} else {
		failReason = r.FailReason
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO invoice_tax_events(invoice_id,event,tax_status_after,tax_no,fail_reason,external_id)
		VALUES($1,'RECEIPT',$2,$3,$4,$5)
		ON CONFLICT (invoice_id,external_id) WHERE external_id <> '' DO NOTHING`,
		id, r.Status, taxNo, failReason, r.ExternalID)
	if err != nil {
		return fmt.Errorf("billing: record tax receipt: %w", err)
	}
	// Only advance states; terminal ISSUED cannot be regressed by delayed failure.
	tag, err := s.db.Exec(ctx, `
		UPDATE invoices SET tax_status = $2, tax_no = $3, tax_fail_reason = $4
		WHERE id = $1 AND tax_status IN ('PENDING','SUBMITTED','FAILED')
		  AND NOT ($2 IN ('FAILED','BLOCKED') AND tax_status = 'ISSUED')`,
		id, r.Status, taxNo, failReason)
	if err != nil {
		return fmt.Errorf("billing: mark tax result: %w", err)
	}
	if tag.RowsAffected() == 0 && r.ExternalID == "" {
		return ErrInvoiceNotTaxable
	}
	return nil
}
