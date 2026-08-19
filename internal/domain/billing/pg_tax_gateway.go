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
// 仅待开/失败/已提交可回填(已开具不可重复回填)。
func (s *PGStore) BackfillTaxNo(ctx context.Context, id int64, taxNo string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE invoices SET tax_status = $2, tax_no = $3, tax_fail_reason = ''
		WHERE id = $1 AND tax_status IN ('PENDING','SUBMITTED','FAILED')`,
		id, TaxStatusIssued, taxNo)
	if err != nil {
		return fmt.Errorf("billing: backfill tax no: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvoiceNotTaxable
	}
	return nil
}

// MarkTaxResult 落税局回执:成功回填票号,失败留痕可重试。
func (s *PGStore) MarkTaxResult(ctx context.Context, id int64, r TaxReceipt) error {
	if r.Status != TaxStatusIssued && r.Status != TaxStatusFailed && r.Status != TaxStatusSUBMITTED {
		return fmt.Errorf("billing: bad receipt status %q", r.Status)
	}
	var taxNo, failReason string
	if r.Status == TaxStatusIssued {
		taxNo = r.TaxNo
	} else {
		failReason = r.FailReason
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE invoices SET tax_status = $2, tax_no = $3, tax_fail_reason = $4
		WHERE id = $1 AND tax_status IN ('PENDING','SUBMITTED','FAILED')`,
		id, r.Status, taxNo, failReason)
	if err != nil {
		return fmt.Errorf("billing: mark tax result: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvoiceNotTaxable
	}
	return nil
}
