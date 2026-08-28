package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// RecordPayment 收款落账:缴费流水 + 账单置 PAID 同事务;pay_no 唯一幂等。
func (s *PGStore) RecordPayment(ctx context.Context, p Payment) (int64, error) {
	r, err := s.RecordPaymentWithCoupon(ctx, p)
	return r.PaymentID, err
}

// resolveBillCustomer 查账单归属客户;账单不存在返回 ErrForeignKeyViolation(防孤儿)。
func (s *PGStore) resolveBillCustomer(ctx context.Context, billID int64) (int64, error) {
	var cust int64
	err := s.db.QueryRow(ctx, `SELECT customer_id FROM bills WHERE id = $1`, billID).Scan(&cust)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("billing: bill %d: %w", billID, ErrForeignKeyViolation)
	}
	if err != nil {
		return 0, fmt.Errorf("billing: resolve bill %d customer: %w", billID, err)
	}
	return cust, nil
}

// RecordPaymentWithCoupon 带券落账:先插全额流水取 id,再同事务行锁核销券,
// 最后把流水金额改写为实收;核销失败整笔回滚(券不可用则缴费不成立)。
// 关联完整性:账单流水必须锚定已存在账单且回填 customer_id(双挂语义,000068);
// 无账单流水(充值/续费)必须带 customer_id 归属,否则落账即孤儿
// (payments 曾 2 条 bill_id/customer_id 双空,audit 2026-08-25)。
func (s *PGStore) RecordPaymentWithCoupon(ctx context.Context, p Payment) (PaymentReceipt, error) {
	if p.Status == "" {
		p.Status = "SUCCESS"
	}
	if !validMethod(p.Method) {
		return PaymentReceipt{}, fmt.Errorf("billing: method %q: %w", p.Method, ErrInvalidMethod)
	}
	if p.PayNo == "" {
		p.PayNo = genPayNo() // 兜底生成:防脏空值撞 pay_no 唯一约束
	}
	if p.BillID > 0 {
		billCust, err := s.resolveBillCustomer(ctx, p.BillID)
		if err != nil {
			return PaymentReceipt{}, err
		}
		if p.CustomerID != 0 && p.CustomerID != billCust {
			return PaymentReceipt{}, fmt.Errorf("billing: payment customer %d mismatch bill customer %d: %w", p.CustomerID, billCust, ErrForeignKeyViolation)
		}
		p.CustomerID = billCust // 强制双挂:落账行 customer_id = 账单客户
	} else if p.CustomerID <= 0 {
		return PaymentReceipt{}, fmt.Errorf("billing: payment needs bill_id or customer_id: %w", ErrForeignKeyViolation)
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
		INSERT INTO payments(pay_no, bill_id, customer_id, amount, method, status, site_name, counter_code, operator_name)
		VALUES($1,NULLIF($2,0),NULLIF($3,0),$4,$5,$6,NULLIF($7,''),NULLIF($8,''),NULLIF($9,'')) RETURNING id`,
		p.PayNo, p.BillID, p.CustomerID, p.Amount, p.Method, p.Status,
		p.SiteName, p.CounterCode, p.OperatorName).Scan(&id)
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
	if p.Status == "SUCCESS" && p.BillID > 0 {
		// 条件置 PAID(纪要 2026-08-28 待定项①,陈磊裁定):累计实收≥应收才置,
		// 未收齐维持原状态,ledger_recon 金额三角自然呈现 PARTIAL;
		// 金额比较按分取整防浮点误差。同事务内刚插流水对本语句可见。
		if _, err := tx.Exec(ctx, `
			UPDATE bills SET status = 'PAID'
			WHERE id = $1 AND status IN ('UNPAID','OVERDUE')
			  AND (COALESCE((SELECT SUM(amount) FROM payments WHERE bill_id = $1 AND status = 'SUCCESS'), 0) * 100 + 0.5)::bigint
			      >= (amount * 100 + 0.5)::bigint`, p.BillID); err != nil {
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
