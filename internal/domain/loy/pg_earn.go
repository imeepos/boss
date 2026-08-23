package loy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// EarnRuleOf 当前生效的缴费送积分规则(最新一条 ENABLED);无则 nil。
func (s *PGStore) EarnRuleOf(ctx context.Context) (*EarnRule, error) {
	var r EarnRule
	err := s.db.QueryRow(ctx,
		`SELECT rule_id, points_per_yuan, min_cents, expire_days, status
		 FROM loy_earn_rules WHERE status='ENABLED' ORDER BY rule_id DESC LIMIT 1`).Scan(
		&r.RuleID, &r.PointsPerYuan, &r.MinCents, &r.ExpireDays, &r.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loy: earn rule of: %w", err)
	}
	return &r, nil
}

// SaveEarnRule 保存缴费送积分规则(插入新行,旧行自动失效),返回新规则 id。
func (s *PGStore) SaveEarnRule(ctx context.Context, r EarnRule) (int64, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("loy: begin earn rule tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE loy_earn_rules SET status='DISABLED' WHERE status='ENABLED'`); err != nil {
		return 0, fmt.Errorf("loy: deactivate earn rules: %w", err)
	}
	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO loy_earn_rules(points_per_yuan, min_cents, expire_days)
		 VALUES($1,$2,$3) RETURNING rule_id`,
		r.PointsPerYuan, r.MinCents, r.ExpireDays).Scan(&id); err != nil {
		return 0, fmt.Errorf("loy: insert earn rule: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("loy: commit earn rule tx: %w", err)
	}
	return id, nil
}

// EarnForPayment 缴费成功送积分:uq_loy_entries_payearn 唯一约束保幂等(重放返回原发放)。
// 未配置规则 / 低于门槛返回 0;金额换算:每 100 分(1 元)积 points_per_yuan 分。
func (s *PGStore) EarnForPayment(ctx context.Context, paymentID, customerID, amountCents int64) (int64, error) {
	rule, err := s.EarnRuleOf(ctx)
	if err != nil || rule == nil || rule.PointsPerYuan <= 0 {
		return 0, err
	}
	if amountCents < rule.MinCents {
		return 0, nil
	}
	points := amountCents / 100 * int64(rule.PointsPerYuan)
	if points <= 0 {
		return 0, nil
	}

	var expiresAt *time.Time
	if rule.ExpireDays > 0 {
		t := time.Now().AddDate(0, 0, rule.ExpireDays)
		expiresAt = &t
	}

	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("loy: begin earn tx: %w", err)
	}
	defer tx.Rollback(ctx)
	after, err := applyDelta(ctx, tx, customerID, points)
	if err != nil {
		return 0, err
	}
	if err := insertEntryExpiring(ctx, tx, customerID, points, after, ReasonPayEarn, paymentID, expiresAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// 唯一约束命中:该缴费已送过,重放返回原发放额。
			var prev int64
			if qerr := s.db.QueryRow(ctx,
				`SELECT delta FROM loy_point_entries WHERE reason=$1 AND ref_id=$2`,
				ReasonPayEarn, paymentID).Scan(&prev); qerr == nil {
				return prev, nil
			}
		}
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("loy: commit earn tx: %w", err)
	}
	return points, nil
}

// RollbackPayment 退款冲销缴费积分:按 PAYMENT_EARN 原额回冲(PAYMENT_REVERSAL 同 ref 幂等)。
// 未送过或已冲销返回 0;积分已被消费导致余额不足时,以负余额冲正记 COMPENSATION 口径不落地,直接返回冲突。
func (s *PGStore) RollbackPayment(ctx context.Context, paymentID, customerID int64) (int64, error) {
	var earned int64
	err := s.db.QueryRow(ctx,
		`SELECT delta FROM loy_point_entries
		 WHERE reason=$1 AND ref_id=$2 AND customer_id=$3 LIMIT 1`,
		ReasonPayEarn, paymentID, customerID).Scan(&earned)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("loy: rollback lookup: %w", err)
	}
	// 已冲销过(同 ref 存在 PAYMENT_REVERSAL)幂等返回 0。
	var exists int
	if err := s.db.QueryRow(ctx,
		`SELECT 1 FROM loy_point_entries WHERE reason=$1 AND ref_id=$2 LIMIT 1`,
		ReasonPayRoll, paymentID).Scan(&exists); err == nil {
		return 0, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("loy: rollback dedup: %w", err)
	}

	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("loy: begin rollback tx: %w", err)
	}
	defer tx.Rollback(ctx)
	after, err := applyDelta(ctx, tx, customerID, -earned)
	if err != nil {
		return 0, err
	}
	if err := insertEntry(ctx, tx, customerID, -earned, after, ReasonPayRoll, paymentID); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("loy: commit rollback tx: %w", err)
	}
	return earned, nil
}

// ExpireDue 过期清算:按客户汇总"到期未过期且未冲销"的获得流水,一次性扣减并标记。
// 单客户批次 = 一条 EXPIRED 负流水(ref=批次内最早 entry_id);余额不足按全部清零口径。
func (s *PGStore) ExpireDue(ctx context.Context) (int, error) {
	rows, err := s.db.Query(ctx, `
		SELECT e.customer_id, SUM(e.delta), MIN(e.entry_id)
		FROM loy_point_entries e
		WHERE e.delta > 0 AND e.expires_at IS NOT NULL
		  AND e.expires_at < now() AND NOT e.expired
		GROUP BY e.customer_id`)
	if err != nil {
		return 0, fmt.Errorf("loy: expire due scan: %w", err)
	}
	type batch struct {
		customer int64
		points   int64
		first    int64
	}
	var batches []batch
	for rows.Next() {
		var b batch
		if err := rows.Scan(&b.customer, &b.points, &b.first); err != nil {
			rows.Close()
			return 0, fmt.Errorf("loy: scan expire batch: %w", err)
		}
		batches = append(batches, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("loy: expire scan err: %w", err)
	}

	for _, b := range batches {
		if err := s.expireBatch(ctx, b.customer, b.points, b.first); err != nil {
			return 0, err
		}
	}
	return len(batches), nil
}

// expireBatch 单客户过期清算事务:标记原流水 + 扣减余额(不足则扣至 0)+
// 记 EXPIRED 负流水(实扣额 = min(批次额, 余额),余额不足部分视为已被消费)。
func (s *PGStore) expireBatch(ctx context.Context, customer, points, firstEntry int64) error {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return fmt.Errorf("loy: begin expire tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE loy_point_entries SET expired=true
		WHERE customer_id=$1 AND delta>0 AND expires_at IS NOT NULL
		  AND expires_at < now() AND NOT expired`, customer); err != nil {
		return fmt.Errorf("loy: mark expired: %w", err)
	}
	var after, bal int64
	if err := tx.QueryRow(ctx,
		`SELECT balance FROM loy_point_ledgers WHERE customer_id=$1 FOR UPDATE`,
		customer).Scan(&bal); err != nil {
		return fmt.Errorf("loy: lock ledger: %w", err)
	}
	deduct := points
	if bal < deduct {
		deduct = bal // 已被消费部分不重复扣
	}
	if deduct > 0 {
		after, err = applyDelta(ctx, tx, customer, -deduct)
		if err != nil {
			return err
		}
		if err := insertEntry(ctx, tx, customer, -deduct, after, ReasonExpired, firstEntry); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("loy: commit expire tx: %w", err)
	}
	return nil
}
