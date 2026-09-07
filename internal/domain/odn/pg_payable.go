package odn

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// payableHead 应付单头查询列(核减/已付为子查询派生,余额由 Go 层计算)。
const payableHead = "cp.id, cp.payable_no, cp.settlement_id, cp.settlement_no, cp.project_id, cp.project_no, " +
	"COALESCE(cp.contractor_id,0), cp.contractor_name, cp.payable_amount, cp.status, COALESCE(cp.void_reason,''), " +
	"to_char(cp.created_at,'YYYY-MM-DD HH24:MI:SS'), to_char(cp.updated_at,'YYYY-MM-DD HH24:MI:SS'), " +
	"(SELECT COALESCE(SUM(amount),0) FROM construction_payable_deductions d WHERE d.payable_id=cp.id), " +
	"(SELECT COALESCE(SUM(amount),0) FROM construction_payable_payments y WHERE y.payable_id=cp.id)"

// scanPayable 扫描应付单头;余额=净应付-已付(VOIDED 冲销后可为负=超付,如实展示)。
func scanPayable(row pgx.Row, p *Payable) error {
	err := row.Scan(&p.ID, &p.PayableNo, &p.SettlementID, &p.SettlementNo, &p.ProjectID, &p.ProjectNo,
		&p.ContractorID, &p.ContractorName, &p.PayableAmount, &p.Status, &p.VoidReason,
		&p.CreatedAt, &p.UpdatedAt, &p.DeductedAmount, &p.PaidAmount)
	if err == nil {
		p.Balance = round2(p.PayableAmount - p.DeductedAmount - p.PaidAmount)
	}
	return err
}

// nextPayableNo 生成应付单号 AP-YYYYMMDD-NNNNN(口径同结算单号,时间戳后 5 位兜底)。
func nextPayableNo() string {
	return fmt.Sprintf("AP-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// nextPaymentNo 生成付款流水号 PAY-YYYYMMDD-NNNNN。
func nextPaymentNo() string {
	return fmt.Sprintf("PAY-%s-%05d", time.Now().Format("20060102"), time.Now().UnixNano()%100000)
}

// ListPayables 应付台账列表(近单优先;status/projectId 过滤)。
func (s *PGStore) ListPayables(ctx context.Context, status string, projectID int64, limit int) ([]Payable, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := "SELECT " + payableHead + " FROM construction_payables cp WHERE 1=1"
	args := []any{}
	if status != "" {
		args = append(args, status)
		q += fmt.Sprintf(" AND cp.status=$%d", len(args))
	}
	if projectID > 0 {
		args = append(args, projectID)
		q += fmt.Sprintf(" AND cp.project_id=$%d", len(args))
	}
	args = append(args, limit)
	q += fmt.Sprintf(" ORDER BY cp.id DESC LIMIT $%d", len(args))
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		log.Printf("[odn-payable] LIST FAILED status=%s proj=%d: %v", status, projectID, err)
		return nil, fmt.Errorf("odn: list payables: %w", err)
	}
	defer rows.Close()
	out := []Payable{}
	for rows.Next() {
		var p Payable
		if err := scanPayable(rows, &p); err != nil {
			return nil, fmt.Errorf("odn: scan payable: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPayable 应付详情聚合(单头+付款/核减/发票流水,单据链回放)。
func (s *PGStore) GetPayable(ctx context.Context, id int64) (*PayableDetail, error) {
	d := &PayableDetail{Payments: []PayablePayment{}, Deductions: []PayableDeduction{}, Invoices: []PayableInvoice{}}
	if err := scanPayable(s.db.QueryRow(ctx, "SELECT "+payableHead+
		" FROM construction_payables cp WHERE cp.id=$1", id), &d.Payable); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("odn: get payable: %w", err)
	}
	rows, err := s.db.Query(ctx, `SELECT id, payable_id, payment_no, amount, method,
		to_char(paid_at,'YYYY-MM-DD HH24:MI:SS'), reference, note, COALESCE(created_by,0),
		to_char(created_at,'YYYY-MM-DD HH24:MI:SS') FROM construction_payable_payments
		WHERE payable_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("odn: list payments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p PayablePayment
		if err := rows.Scan(&p.ID, &p.PayableID, &p.PaymentNo, &p.Amount, &p.Method, &p.PaidAt,
			&p.Reference, &p.Note, &p.CreatedBy, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan payment: %w", err)
		}
		d.Payments = append(d.Payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("odn: iterate payments: %w", err)
	}
	rows2, err := s.db.Query(ctx, `SELECT id, payable_id, amount, reason, COALESCE(created_by,0),
		to_char(created_at,'YYYY-MM-DD HH24:MI:SS') FROM construction_payable_deductions
		WHERE payable_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("odn: list deductions: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var dd PayableDeduction
		if err := rows2.Scan(&dd.ID, &dd.PayableID, &dd.Amount, &dd.Reason, &dd.CreatedBy, &dd.CreatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan deduction: %w", err)
		}
		d.Deductions = append(d.Deductions, dd)
	}
	if err := rows2.Err(); err != nil {
		return nil, fmt.Errorf("odn: iterate deductions: %w", err)
	}
	rows3, err := s.db.Query(ctx, `SELECT id, payable_id, invoice_no, amount,
		COALESCE(to_char(invoiced_at,'YYYY-MM-DD'),''), note, COALESCE(created_by,0),
		to_char(created_at,'YYYY-MM-DD HH24:MI:SS') FROM construction_payable_invoices
		WHERE payable_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, fmt.Errorf("odn: list invoices: %w", err)
	}
	defer rows3.Close()
	for rows3.Next() {
		var iv PayableInvoice
		if err := rows3.Scan(&iv.ID, &iv.PayableID, &iv.InvoiceNo, &iv.Amount, &iv.InvoicedAt,
			&iv.Note, &iv.CreatedBy, &iv.CreatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan invoice: %w", err)
		}
		d.Invoices = append(d.Invoices, iv)
	}
	return d, rows3.Err()
}

// payableSums 事务内读核减/已付合计(FOR UPDATE 行锁后调用,防并发超登记)。
func payableSums(ctx context.Context, q pgx.Tx, id int64) (payable, deducted, paid float64, status string, err error) {
	err = q.QueryRow(ctx, `SELECT payable_amount, status,
		(SELECT COALESCE(SUM(amount),0) FROM construction_payable_deductions WHERE payable_id=$1),
		(SELECT COALESCE(SUM(amount),0) FROM construction_payable_payments WHERE payable_id=$1)
		FROM construction_payables WHERE id=$1 FOR UPDATE`, id).
		Scan(&payable, &status, &deducted, &paid)
	return
}

// RegisterPayablePayment 付款流水登记:VOIDED 拒绝;单笔不超未付余额(部分付款/分期逐笔登记);
// 状态按派生口径推进 OPEN/PARTIAL→PAID。
func (s *PGStore) RegisterPayablePayment(ctx context.Context, payableID, accountID int64, amount float64,
	method, paidAt, reference, note string) (*PayablePayment, error) {
	if len(note) > 255 || len(reference) > 64 {
		return nil, fmt.Errorf("odn: payment note/reference too long: %w", ErrInvalidInput)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-payable] PAY TX BEGIN FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: pay begin: %w", err)
	}
	defer tx.Rollback(ctx)
	payable, deducted, paid, status, err := payableSums(ctx, tx, payableID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		log.Printf("[odn-payable] PAY READ FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: pay read: %w", err)
	}
	if status == APVoided {
		return nil, fmt.Errorf("odn: payable %d VOIDED: %w", payableID, ErrPayableState)
	}
	if err := ValidatePayRegister(method, amount, payable-deducted-paid); err != nil {
		if errors.Is(err, ErrPayableState) {
			return nil, fmt.Errorf("odn: payable %d 余额不足 balance=%f: %w", payableID, round2(payable-deducted-paid), err)
		}
		return nil, fmt.Errorf("odn: payable %d pay amount=%f method=%s: %w", payableID, amount, method, err)
	}
	no := nextPaymentNo()
	p := PayablePayment{PayableID: payableID, PaymentNo: no, Amount: round2(amount),
		Method: method, Reference: reference, Note: note, CreatedBy: accountID}
	err = tx.QueryRow(ctx, `INSERT INTO construction_payable_payments
		(payable_id, payment_no, amount, method, paid_at, reference, note, created_by)
		VALUES ($1,$2,$3,$4,COALESCE($5::timestamptz,now()),$6,$7,$8)
		RETURNING id, to_char(paid_at,'YYYY-MM-DD HH24:MI:SS'), to_char(created_at,'YYYY-MM-DD HH24:MI:SS')`,
		payableID, no, p.Amount, method, nilIfEmpty(paidAt), reference, note, accountID).
		Scan(&p.ID, &p.PaidAt, &p.CreatedAt)
	if err != nil {
		log.Printf("[odn-payable] PAY REGISTER FAILED id=%d amount=%f: %v", payableID, amount, err)
		return nil, fmt.Errorf("odn: pay register: %w", err)
	}
	newStatus := ComputePayableStatus(payable, deducted, paid+p.Amount)
	if _, err := tx.Exec(ctx, `UPDATE construction_payables SET status=$2, updated_at=now() WHERE id=$1`,
		payableID, newStatus); err != nil {
		log.Printf("[odn-payable] STATUS UPDATE FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: payable status update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-payable] PAY COMMIT FAILED id=%d no=%s: %v", payableID, no, err)
		return nil, fmt.Errorf("odn: pay commit: %w", err)
	}
	return &p, nil
}

// DeductPayable 核减:append-only 明细(原因必填留痕);核减不超应付,核减后净应付不得低于已付;
// 核减后应付状态同步(派生口径重算)。
func (s *PGStore) DeductPayable(ctx context.Context, payableID, accountID int64, amount float64, reason string) (*PayableDeduction, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("[odn-payable] DEDUCT TX BEGIN FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: deduct begin: %w", err)
	}
	defer tx.Rollback(ctx)
	payable, deducted, paid, status, err := payableSums(ctx, tx, payableID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		log.Printf("[odn-payable] DEDUCT READ FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: deduct read: %w", err)
	}
	if status == APVoided {
		return nil, fmt.Errorf("odn: payable %d VOIDED: %w", payableID, ErrPayableState)
	}
	if err := ValidateDeduct(reason, amount, payable, deducted, paid); err != nil {
		if errors.Is(err, ErrPayableState) {
			return nil, fmt.Errorf("odn: payable %d 核减超限 payable=%f deducted=%f paid=%f amount=%f: %w",
				payableID, payable, deducted, paid, amount, err)
		}
		return nil, fmt.Errorf("odn: payable %d deduct amount=%f: %w", payableID, amount, err)
	}
	d := PayableDeduction{PayableID: payableID, Amount: round2(amount), Reason: reason, CreatedBy: accountID}
	err = tx.QueryRow(ctx, `INSERT INTO construction_payable_deductions (payable_id, amount, reason, created_by)
		VALUES ($1,$2,$3,$4) RETURNING id, to_char(created_at,'YYYY-MM-DD HH24:MI:SS')`,
		payableID, d.Amount, reason, accountID).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		log.Printf("[odn-payable] DEDUCT REGISTER FAILED id=%d amount=%f: %v", payableID, amount, err)
		return nil, fmt.Errorf("odn: deduct register: %w", err)
	}
	newStatus := ComputePayableStatus(payable, deducted+d.Amount, paid)
	if _, err := tx.Exec(ctx, `UPDATE construction_payables SET status=$2, updated_at=now() WHERE id=$1`,
		payableID, newStatus); err != nil {
		log.Printf("[odn-payable] DEDUCT STATUS FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: payable status update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("[odn-payable] DEDUCT COMMIT FAILED id=%d: %v", payableID, err)
		return nil, fmt.Errorf("odn: deduct commit: %w", err)
	}
	return &d, nil
}

// RegisterPayableInvoice 发票登记(纯登记;VOIDED 拒绝;同应付内发票号唯一)。
func (s *PGStore) RegisterPayableInvoice(ctx context.Context, payableID, accountID int64, amount float64,
	invoiceNo, invoicedAt, note string) (*PayableInvoice, error) {
	if invoiceNo == "" || amount <= 0 || len(note) > 255 {
		return nil, fmt.Errorf("odn: invoice input invalid: %w", ErrInvalidInput)
	}
	var status string
	if err := s.db.QueryRow(ctx, `SELECT status FROM construction_payables WHERE id=$1`, payableID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("odn: invoice payable read: %w", err)
	}
	if status == APVoided {
		return nil, fmt.Errorf("odn: payable %d VOIDED: %w", payableID, ErrPayableState)
	}
	inv := PayableInvoice{PayableID: payableID, InvoiceNo: invoiceNo, Amount: round2(amount), Note: note, CreatedBy: accountID}
	err := s.db.QueryRow(ctx, `INSERT INTO construction_payable_invoices
		(payable_id, invoice_no, amount, invoiced_at, note, created_by)
		VALUES ($1,$2,$3,COALESCE($4::date,NULL),$5,$6)
		RETURNING id, COALESCE(to_char(invoiced_at,'YYYY-MM-DD'),''), to_char(created_at,'YYYY-MM-DD HH24:MI:SS')`,
		payableID, invoiceNo, inv.Amount, nilIfEmpty(invoicedAt), note, accountID).
		Scan(&inv.ID, &inv.InvoicedAt, &inv.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("odn: payable %d 发票 %s 重复登记: %w", payableID, invoiceNo, ErrPayableState)
		}
		log.Printf("[odn-payable] INVOICE REGISTER FAILED id=%d no=%s: %v", payableID, invoiceNo, err)
		return nil, fmt.Errorf("odn: invoice register: %w", err)
	}
	return &inv, nil
}

// nilIfEmpty 空串转 NULL(时间/日期可空入参统一)。
func nilIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
