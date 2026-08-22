package promotion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 最小数据库接口;*pgxpool.Pool 与 pgx.Tx 均满足,测试可注入 mock。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// beginner 开事务能力(核销与 billing 同事务)。
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore Service 的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// randCode 生成随机码(前缀+12 hex),券号与兑换/转赠码共用此生成器。
func randCode(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// isNoRows pgx 无行判定简写。
func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// execAffected 执行并要求命中 1 行,否则 ErrNotFound。
func (s *PGStore) execAffected(ctx context.Context, op, sql string, args ...any) error {
	tag, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("promotion: %s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeductForPayment 缴费核销:在调用方事务(billing.RecordPayment 同事务)内
// 行锁占用券并写核销记录,返回实际抵扣额(分)。券不可用/一券多用 → ErrConflict。
func (s *PGStore) DeductForPayment(ctx context.Context, tx pgx.Tx, couponID string, customerID, paymentID, billCents int64) (int64, error) {
	var cType string
	var face, threshold, maxDisc int64
	err := tx.QueryRow(ctx, `
		UPDATE coupons SET status='USED', used_at=now(), payment_id=$3, code=NULL
		WHERE coupon_id=$1 AND customer_id=$2 AND status='ISSUED' AND code IS NULL
		  AND (expire_at IS NULL OR expire_at > now())
		RETURNING type, COALESCE(face_value, amount), threshold, COALESCE(max_discount,0)`,
		couponID, customerID, paymentID).Scan(&cType, &face, &threshold, &maxDisc)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, &conflictError{reason: "券不可用或已被使用: " + couponID}
		}
		return 0, fmt.Errorf("promotion: redeem coupon: %w", err)
	}
	deducted := deductAmount(cType, face, threshold, maxDisc, billCents)
	if deducted < 0 {
		return 0, &conflictError{reason: "未达使用门槛: " + couponID}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO coupon_redemptions(coupon_id, payment_id, customer_id, deducted_amount)
		VALUES($1,$2,$3,$4)`, couponID, paymentID, customerID, deducted); err != nil {
		return 0, fmt.Errorf("promotion: insert redemption: %w", err)
	}
	return deducted, nil
}

// deductAmount 按券类型计算抵扣额(分);未达门槛返回 -1。
func deductAmount(cType string, face, threshold, maxDisc, billCents int64) int64 {
	switch cType {
	case TypeFullCut:
		if billCents < threshold {
			return -1
		}
		return min(face, billCents)
	case TypeDiscout:
		d := billCents - billCents*face/10000
		if maxDisc > 0 && d > maxDisc {
			d = maxDisc
		}
		return d
	default: // CASH
		return min(face, billCents)
	}
}
