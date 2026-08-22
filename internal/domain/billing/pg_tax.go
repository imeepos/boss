package billing

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
)

// VATRate 默认增值税率(TAX-001:应税默认 12%;免税/跨境后续经税务组配置扩展)。
const VATRate = 0.12

// arnDocType 发票序列;RECEIPT 序列已建表内置,收据出具时复用同一发号机制。
const arnDocType = "INVOICE"

// beginner 显式事务入口;*pgxpool.Pool 与 pgxmock 均满足。
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// round2 分位两位小数四舍五入(GEN-006:账单/收据/发票/退款同一规则)。
func round2(v float64) float64 { return math.Round(v*100) / 100 }

const invoiceCols = `id, invoice_no, bill_id, bill_no, customer_id, customer_name, title,
	net_amount, vat_rate, vat_amount, total_amount, status, void_reason,
	tax_jurisdiction, tax_channel, tax_status, tax_no, tax_fail_reason, issued_at, voided_at`

func scanInvoice(row pgx.Row) (*Invoice, error) {
	var inv Invoice
	err := row.Scan(&inv.ID, &inv.InvoiceNo, &inv.BillID, &inv.BillNo, &inv.CustomerID, &inv.CustomerName,
		&inv.Title, &inv.NetAmount, &inv.VatRate, &inv.VatAmount, &inv.TotalAmount,
		&inv.Status, &inv.VoidReason,
		&inv.TaxJurisdiction, &inv.TaxChannel, &inv.TaxStatus, &inv.TaxNo, &inv.TaxFailReason,
		&inv.IssuedAt, &inv.VoidedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListInvoices 列出发票;customerID=0 返回全部,按开票时间倒序。
func (s *PGStore) ListInvoices(ctx context.Context, customerID int64) ([]Invoice, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+invoiceCols+` FROM invoices WHERE ($1::bigint = 0 OR customer_id = $1) ORDER BY id DESC`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list invoices: %w", err)
	}
	defer rows.Close()
	out := make([]Invoice, 0)
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, fmt.Errorf("billing: scan invoice: %w", err)
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

// IssueInvoicesForPeriod 出账完成后自动开票(CT-007):逐张独立事务,失败进异常清单。
func (s *PGStore) IssueInvoicesForPeriod(ctx context.Context, period string) (InvoiceRunResult, error) {
	ids, err := s.uninvoicedBillIDs(ctx, period)
	if err != nil {
		return InvoiceRunResult{}, err
	}
	res := InvoiceRunResult{FailedIDs: []int64{}}
	for _, id := range ids {
		if _, err := s.issueInvoiceForBillByID(ctx, id); err != nil {
			if err == ErrDuplicateInvoice { // 并发下他人已开,幂等跳过
				continue
			}
			res.FailedIDs = append(res.FailedIDs, id)
			continue
		}
		res.Issued++
	}
	return res, nil
}

// uninvoicedBillIDs 该账期尚无在发票的账单(开票批次输入)。
func (s *PGStore) uninvoicedBillIDs(ctx context.Context, period string) ([]int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT b.id FROM bills b
		WHERE b.period = $1 AND NOT EXISTS
			(SELECT 1 FROM invoices i WHERE i.bill_id = b.id AND i.status = 'ISSUED')
		ORDER BY b.id`, period)
	if err != nil {
		return nil, fmt.Errorf("billing: list uninvoiced bills: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("billing: scan bill id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// IssueInvoiceForBill 门户按单开票:customer+billNo 定位账单(归属校验),
// 已开票幂等返回已有票;未命中/非归属返回 ErrNotFound。
func (s *PGStore) IssueInvoiceForBill(ctx context.Context, customerID int64, billNo string) (*Invoice, error) {
	var billID int64
	err := s.db.QueryRow(ctx,
		`SELECT id FROM bills WHERE customer_id = $1 AND bill_no = $2`, customerID, billNo).Scan(&billID)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: locate bill: %w", err)
	}
	inv, err := s.issueInvoiceForBillByID(ctx, billID)
	if err == ErrDuplicateInvoice { // 并发下他人已开,回读已有票
		return s.activeInvoiceOfBill(ctx, billID)
	}
	return inv, err
}

// activeInvoiceOfBill 账单当前在发票(非 VOIDED,最新一张)。
func (s *PGStore) activeInvoiceOfBill(ctx context.Context, billID int64) (*Invoice, error) {
	inv, err := scanInvoice(s.db.QueryRow(ctx,
		`SELECT `+invoiceCols+` FROM invoices WHERE bill_id = $1 AND status != 'VOIDED'
		ORDER BY id DESC LIMIT 1`, billID))
	if err == pgx.ErrNoRows {
		return nil, ErrDuplicateInvoice
	}
	if err != nil {
		return nil, fmt.Errorf("billing: load active invoice: %w", err)
	}
	return inv, nil
}

// issueInvoiceForBillByID 为单张账单开票:显式事务内 查重→占号→插入,回滚号回退。
func (s *PGStore) issueInvoiceForBillByID(ctx context.Context, billID int64) (*Invoice, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("billing: begin issue tx: %w", err)
	}
	defer tx.Rollback(ctx)
	inv, err := issueInvoiceTx(ctx, tx, billID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("billing: commit issue tx: %w", err)
	}
	return inv, nil
}

// issueInvoiceTx 事务内开票:先查重(幂等,不耗号)→取账单净额→ARN 占号→插入发票。
func issueInvoiceTx(ctx context.Context, tx pgx.Tx, billID int64) (*Invoice, error) {
	var dup int
	err := tx.QueryRow(ctx,
		`SELECT 1 FROM invoices WHERE bill_id = $1 AND status = 'ISSUED'`, billID).Scan(&dup)
	if err == nil {
		return nil, ErrDuplicateInvoice
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("billing: dup check: %w", err)
	}
	var b Bill
	err = tx.QueryRow(ctx,
		`SELECT id, bill_no, customer_id, customer_name, amount FROM bills WHERE id = $1`, billID).
		Scan(&b.BillID, &b.BillNo, &b.CustomerID, &b.CustomerName, &b.Amount)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: load bill: %w", err)
	}
	arn, err := occupyArn(ctx, tx)
	if err != nil {
		return nil, err
	}
	vat := round2(b.Amount * VATRate)
	inv := Invoice{
		InvoiceNo: arn, BillID: b.BillID, BillNo: b.BillNo, CustomerID: b.CustomerID,
		CustomerName: b.CustomerName, Title: b.CustomerName, NetAmount: round2(b.Amount),
		VatRate: VATRate, VatAmount: vat, TotalAmount: round2(b.Amount + vat), Status: "ISSUED",
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices(invoice_no, bill_id, bill_no, customer_id, customer_name, title,
			net_amount, vat_rate, vat_amount, total_amount)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, issued_at`,
		inv.InvoiceNo, inv.BillID, inv.BillNo, inv.CustomerID, inv.CustomerName, inv.Title,
		inv.NetAmount, inv.VatRate, inv.VatAmount, inv.TotalAmount).Scan(&inv.ID, &inv.IssuedAt)
	if err != nil {
		return nil, fmt.Errorf("billing: insert invoice: %w", err)
	}
	return &inv, nil
}

// occupyArn 原子占号:行锁串行(TAX-004),与插入同事务,回滚号回退(无跳号)。
func occupyArn(ctx context.Context, tx pgx.Tx) (string, error) {
	var n int64
	var prefix string
	err := tx.QueryRow(ctx, `
		UPDATE arn_sequences SET next_no = next_no + 1, updated_at = now()
		WHERE doc_type = $1 RETURNING next_no - 1, prefix`, arnDocType).Scan(&n, &prefix)
	if err != nil {
		return "", fmt.Errorf("billing: occupy arn: %w", err)
	}
	return fmt.Sprintf("%s%08d", prefix, n), nil
}

// VoidInvoice 作废发票:编号保留不回收(TAX-003/004),仅 ISSUED 可作废。
func (s *PGStore) VoidInvoice(ctx context.Context, id int64, reason string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE invoices SET status = 'VOIDED', void_reason = $2, voided_at = now()
		WHERE id = $1 AND status = 'ISSUED'`, id, reason)
	if err != nil {
		return fmt.Errorf("billing: void invoice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIllegalInvoiceTransition
	}
	return nil
}

// ReissueInvoice 重开:原票 VOID 保留编号 + 新票新 ARN(与作废同事务)。
func (s *PGStore) ReissueInvoice(ctx context.Context, id int64) (*Invoice, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("billing: begin reissue tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var billID int64
	err = tx.QueryRow(ctx, `SELECT bill_id FROM invoices WHERE id = $1 FOR UPDATE`, id).Scan(&billID)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: load invoice: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE invoices SET status = 'VOIDED', void_reason = 'reissue', voided_at = now()
		WHERE id = $1 AND status = 'ISSUED'`, id)
	if err != nil {
		return nil, fmt.Errorf("billing: void for reissue: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrIllegalInvoiceTransition
	}
	inv, err := issueInvoiceTx(ctx, tx, billID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("billing: commit reissue tx: %w", err)
	}
	return inv, nil
}
