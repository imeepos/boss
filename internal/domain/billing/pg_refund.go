package billing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrPaymentNotRefundable 流水不可退款(非 SUCCESS/账单还有其他在流水/已退)。
var ErrPaymentNotRefundable = errors.New("billing: payment not refundable")

// RefundPayment 全额退款(000112):事务内行锁流水→置 REFUNDED(留痕 reason/时间)
// →账单无其他 SUCCESS 流水时回 UNPAID(重新可收款,幂等重收款)。
// 充值流水(bill_id NULL)退款只置流水,不动账单。发票不自动作废(人工经 void/reissue)。
func (s *PGStore) RefundPayment(ctx context.Context, paymentID int64, reason string) (*Payment, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("billing: begin refund tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var billID int64
	// 000068 起 bill_id 可空(充值类流水),必须 COALESCE 成 0 再 Scan(int64),
	// 否则充值退款即 "cannot scan NULL into *int64"(102 验收暴露)。
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(bill_id, 0) FROM payments WHERE id = $1 FOR UPDATE`, paymentID).Scan(&billID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: lock payment: %w", err)
	}
	if err := refundPaymentRow(ctx, tx, paymentID, reason); err != nil {
		return nil, err
	}
	if billID > 0 {
		if err := reopenBillIfNoPaidFlow(ctx, tx, billID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("billing: commit refund tx: %w", err)
	}
	out, err := s.getPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// refundPaymentRow 流水置 REFUNDED;仅 SUCCESS 可退(重复退/失败流水被拒)。
func refundPaymentRow(ctx context.Context, tx pgx.Tx, paymentID int64, reason string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE payments SET status = 'REFUNDED', refund_reason = $2, refunded_at = now()
		WHERE id = $1 AND status = 'SUCCESS'`, paymentID, reason)
	if err != nil {
		return fmt.Errorf("billing: refund payment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPaymentNotRefundable
	}
	return nil
}

// reopenBillIfNoPaidFlow 账单回 UNPAID 前校验:退掉本笔后不得仍有 SUCCESS 流水
// (多笔合并缴款的账单须先逐笔处理,避免部分退款语义混入全额模型)。
func reopenBillIfNoPaidFlow(ctx context.Context, tx pgx.Tx, billID int64) error {
	tag, err := tx.Exec(ctx, `
		UPDATE bills SET status = 'UNPAID'
		WHERE id = $1 AND NOT EXISTS
			(SELECT 1 FROM payments WHERE bill_id = $1 AND status = 'SUCCESS')`, billID)
	if err != nil {
		return fmt.Errorf("billing: reopen bill: %w", err)
	}
	_ = tag // 账单可能因其他在流水保持 PAID,0 行合法
	return nil
}

// getPayment 回读退款后流水(含留痕列);refunded_at 经 sql.NullTime 兼容 NULL。
func (s *PGStore) getPayment(ctx context.Context, paymentID int64) (*Payment, error) {
	var p Payment
	var refundedAt sql.NullTime
	err := s.db.QueryRow(ctx, `
		SELECT id, pay_no, COALESCE(bill_id,0), amount, method, status, refund_reason, refunded_at
		FROM payments WHERE id = $1`, paymentID).
		Scan(&p.ID, &p.PayNo, &p.BillID, &p.Amount, &p.Method, &p.Status, &p.RefundReason, &refundedAt)
	if err != nil {
		return nil, fmt.Errorf("billing: reload payment: %w", err)
	}
	if refundedAt.Valid {
		p.RefundedAt = &refundedAt.Time
	}
	return &p, nil
}
