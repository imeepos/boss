package loy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore Service 的 PostgreSQL 实现;兑换的价查/发券经构造注入,不依赖 promotion 包。
type PGStore struct {
	db    dbtx
	price PriceOf
	issue IssueCoupon
}

// NewPGStore 构造;price/issue 由 app 装配绑定 promotion 实现。
func NewPGStore(db dbtx, price PriceOf, issue IssueCoupon) *PGStore {
	return &PGStore{db: db, price: price, issue: issue}
}

// Balance 余额(无账本视为 0)。
func (s *PGStore) Balance(ctx context.Context, customerID int64) (int64, error) {
	var bal int64
	err := s.db.QueryRow(ctx,
		`SELECT balance FROM loy_point_ledgers WHERE customer_id=$1`, customerID).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("loy: balance: %w", err)
	}
	return bal, nil
}

// Entries 流水(近 100 条)。
func (s *PGStore) Entries(ctx context.Context, customerID int64) ([]Entry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT entry_id, customer_id, delta, balance_after, reason, COALESCE(ref_id,0),
			COALESCE(to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),'')
		FROM loy_point_entries WHERE customer_id=$1 ORDER BY entry_id DESC LIMIT 100`, customerID)
	if err != nil {
		return nil, fmt.Errorf("loy: entries: %w", err)
	}
	defer rows.Close()
	out := make([]Entry, 0)
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.EntryID, &e.CustomerID, &e.Delta, &e.BalanceAfter,
			&e.Reason, &e.RefID, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("loy: scan entry: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Adjust 手动调整:账本 upsert + 流水同事务;扣减不得为负。
func (s *PGStore) Adjust(ctx context.Context, customerID int64, delta int64, reason string) (int64, error) {
	if delta == 0 {
		return 0, ErrConflict
	}
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("loy: begin adjust tx: %w", err)
	}
	defer tx.Rollback(ctx)
	after, err := applyDelta(ctx, tx, customerID, delta)
	if err != nil {
		return 0, err
	}
	if err := insertEntry(ctx, tx, customerID, delta, after, reason, 0); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("loy: commit adjust tx: %w", err)
	}
	return after, nil
}

// applyDelta 账本余额增减(行锁 + 非负约束),返回新余额;不足返回 ErrConflict。
func applyDelta(ctx context.Context, tx pgx.Tx, customerID, delta int64) (int64, error) {
	var after int64
	err := tx.QueryRow(ctx, `
		INSERT INTO loy_point_ledgers(customer_id, balance) VALUES($1, 0)
		ON CONFLICT (customer_id) DO NOTHING`, customerID).Scan()
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("loy: ensure ledger: %w", err)
	}
	err = tx.QueryRow(ctx, `
		UPDATE loy_point_ledgers SET balance = balance + $2, updated_at = now()
		WHERE customer_id = $1 AND balance + $2 >= 0
		RETURNING balance`, customerID, delta).Scan(&after)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("%w: 积分不足", ErrConflict)
	}
	if err != nil {
		return 0, fmt.Errorf("loy: apply delta: %w", err)
	}
	return after, nil
}

// insertEntry 流水落库(无有效期)。
func insertEntry(ctx context.Context, tx pgx.Tx, customerID, delta, after int64, reason string, refID int64) error {
	return insertEntryExpiring(ctx, tx, customerID, delta, after, reason, refID, nil)
}

// insertEntryExpiring 带有效期流水落库(expiresAt nil=不限定)。
func insertEntryExpiring(ctx context.Context, tx pgx.Tx, customerID, delta, after int64,
	reason string, refID int64, expiresAt *time.Time) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO loy_point_entries(customer_id, delta, balance_after, reason, ref_id, expires_at)
		VALUES($1,$2,$3,$4,NULLIF($5,0),$6)`, customerID, delta, after, reason, refID, expiresAt); err != nil {
		return fmt.Errorf("loy: insert entry: %w", err)
	}
	return nil
}

// Exchange 积分换券:价查 → 扣积分(独立事务) → 发券;发券失败补偿回补。
// 跨域非同事务,补偿模式见 adopted note 2026-08-26。
func (s *PGStore) Exchange(ctx context.Context, customerID, templateID int64) (string, int64, error) {
	price, err := s.price(ctx, templateID)
	if err != nil {
		return "", 0, err
	}
	if price <= 0 {
		return "", 0, fmt.Errorf("%w: 该券模板不可积分兑换", ErrConflict)
	}

	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return "", 0, fmt.Errorf("loy: begin exchange tx: %w", err)
	}
	defer tx.Rollback(ctx)
	after, err := applyDelta(ctx, tx, customerID, -price)
	if err != nil {
		return "", 0, err
	}
	if err := insertEntry(ctx, tx, customerID, -price, after, ReasonExchange, templateID); err != nil {
		return "", 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, fmt.Errorf("loy: commit exchange tx: %w", err)
	}

	couponID, err := s.issue(ctx, templateID, customerID)
	if err != nil {
		_, _ = s.Adjust(ctx, customerID, price, ReasonReversal)
		return "", 0, err
	}
	return couponID, price, nil
}
