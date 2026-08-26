package billing

// RecordTopup 充值落账:无账单缴费流水 + 余额增加同事务(pay_no 唯一约束幂等)。
// 余额权威态在 portal 域 portal_wallets(裁定 D1);本方法在充值记账事务内一并更新,
// 与 userdata.AdjustUserBalance 直写 portal_wallets 同型(薄适配),保证"流水+余额"原子,
// 修复 webhook 无账单充值只落流水不增余额的缺口(2026-08-30 收口)。
// 失败/拒绝路径(p.Status=FAILED)只落 FAILED 流水,不动余额。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrPayNoExists 同 pay_no 流水已存在(渠道重投幂等信号,调用方按已落账处理)。
var ErrPayNoExists = errors.New("billing: pay_no already exists")

// RecordTopup 充值落账:插入无账单流水(必须带 customer_id,防孤儿)并原子增余额。
func (s *PGStore) RecordTopup(ctx context.Context, p Payment) (int64, error) {
	if p.Status == "" {
		p.Status = "SUCCESS"
	}
	if p.BillID > 0 {
		return 0, fmt.Errorf("billing: topup must be bill-less: %w", ErrForeignKeyViolation)
	}
	if p.CustomerID == 0 {
		return 0, fmt.Errorf("billing: topup needs customer_id: %w", ErrForeignKeyViolation)
	}
	if p.CustomerID > 0 {
		ok, err := s.exists(ctx, "customers", p.CustomerID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("billing: customer %d: %w", p.CustomerID, ErrForeignKeyViolation)
		}
	}
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("billing: begin topup tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO payments(pay_no, bill_id, customer_id, amount, method, status)
		VALUES($1, NULL, $2, $3, $4, $5) RETURNING id`,
		p.PayNo, p.CustomerID, p.Amount, p.Method, p.Status).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrPayNoExists
		}
		return 0, fmt.Errorf("billing: insert topup payment: %w", err)
	}
	if p.Status == "SUCCESS" {
		// 充值成功:余额入账(幂等由 pay_no 唯一约束兜底,事务回滚不产生半状态)。
		if _, err := tx.Exec(ctx, `
			INSERT INTO portal_wallets(customer_id, balance) VALUES($1, $2)
			ON CONFLICT (customer_id) DO UPDATE SET balance = portal_wallets.balance + EXCLUDED.balance`,
			p.CustomerID, p.Amount); err != nil {
			return 0, fmt.Errorf("billing: credit topup balance: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("billing: commit topup tx: %w", err)
	}
	return id, nil
}

// PaymentExistsByPayNo 同 pay_no 流水是否存在(webhook 重投幂等直查,替代全表扫描)。
func (s *PGStore) PaymentExistsByPayNo(ctx context.Context, payNo string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM payments WHERE pay_no = $1)`, payNo).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("billing: check payment %s: %w", payNo, err)
	}
	return ok, nil
}

// isUniqueViolation pg 唯一约束冲突(23505)判定。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
