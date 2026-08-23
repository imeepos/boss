package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// RecordPayment 收款落账:缴费流水 + 账单置 PAID 同事务;pay_no 唯一幂等。
func (s *PGStore) RecordPayment(ctx context.Context, p Payment) (int64, error) {
	r, err := s.RecordPaymentWithCoupon(ctx, p)
	return r.PaymentID, err
}

// RecordPaymentWithCoupon 带券落账:先插全额流水取 id,再同事务行锁核销券,
// 最后把流水金额改写为实收;核销失败整笔回滚(券不可用则缴费不成立)。
func (s *PGStore) RecordPaymentWithCoupon(ctx context.Context, p Payment) (PaymentReceipt, error) {
	if p.Status == "" {
		p.Status = "SUCCESS"
	}
	if p.CouponID != "" && s.couponDeduc == nil {
		return PaymentReceipt{}, fmt.Errorf("billing: coupon deductor not configured")
	}
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return PaymentReceipt{}, fmt.Errorf("billing: begin payment tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO payments(pay_no, bill_id, amount, method, status)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		p.PayNo, p.BillID, p.Amount, p.Method, p.Status).Scan(&id)
	if err != nil {
		return PaymentReceipt{}, fmt.Errorf("billing: insert payment: %w", err)
	}
	deducted := int64(0)
	if p.CouponID != "" {
		deducted, err = s.redeemCoupon(ctx, tx, p, id)
		if err != nil {
			return PaymentReceipt{}, err
		}
		if deducted > 0 {
			if _, err := tx.Exec(ctx, `UPDATE payments SET amount = amount - $2 WHERE id = $1`,
				id, float64(deducted)/100); err != nil {
				return PaymentReceipt{}, fmt.Errorf("billing: adjust payment amount: %w", err)
			}
		}
	}
	if p.Status == "SUCCESS" {
		// OVERDUE 账单缴清同样置 PAID(dunning 置逾期后缴费闭环)。
		if _, err := tx.Exec(ctx,
			`UPDATE bills SET status = 'PAID' WHERE id = $1 AND status IN ('UNPAID','OVERDUE')`, p.BillID); err != nil {
			return PaymentReceipt{}, fmt.Errorf("billing: mark bill paid: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PaymentReceipt{}, fmt.Errorf("billing: commit payment tx: %w", err)
	}
	return PaymentReceipt{PaymentID: id, Amount: p.Amount - float64(deducted)/100, DeductedCents: deducted}, nil
}

// redeemCoupon 券核销:解析缴费归属客户(账单兜底),单位 元→分 后调注入核销器。
func (s *PGStore) redeemCoupon(ctx context.Context, tx pgx.Tx, p Payment, paymentID int64) (int64, error) {
	cust := p.CustomerID
	if cust == 0 && p.BillID > 0 {
		if err := tx.QueryRow(ctx,
			`SELECT customer_id FROM bills WHERE id = $1`, p.BillID).Scan(&cust); err != nil {
			return 0, fmt.Errorf("billing: resolve bill customer: %w", err)
		}
	}
	if cust == 0 {
		return 0, fmt.Errorf("billing: coupon payment %d: %w", p.CustomerID, ErrForeignKeyViolation)
	}
	billCents := int64(p.Amount*100 + 0.5)
	deducted, err := s.couponDeduc(ctx, tx, p.CouponID, cust, paymentID, billCents)
	if err != nil {
		return 0, fmt.Errorf("billing: redeem coupon %s: %w", p.CouponID, err)
	}
	return deducted, nil
}
