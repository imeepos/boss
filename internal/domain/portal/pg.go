package portal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/ymm-001/boss/internal/pkg/sms"
)

// pgStore 门户状态 PG 实现。
type pgStore struct {
	pool   *pgxpool.Pool
	sender sms.Sender // 验证码外发通道;nil 表示不外发(仅落库,旧测试路径)
}

// NewPGStore 构造门户状态存储(不外发验证码,兼容旧装配)。
func NewPGStore(pool *pgxpool.Pool) Service { return &pgStore{pool: pool} }

// NewPGStoreWithSender 构造带短信通道的存储:IssueSms 落库后真实外发。
func NewPGStoreWithSender(pool *pgxpool.Pool, sender sms.Sender) Service {
	return &pgStore{pool: pool, sender: sender}
}

func (s *pgStore) IssueSms(ctx context.Context, phone, scene string) error {
	// 冷却:同 phone+scene 60s 内重复签发直接拒绝,防刷。
	var recent bool
	err := s.pool.QueryRow(ctx,
		`SELECT issued_at > now() - interval '60 seconds' FROM portal_sms_codes WHERE phone=$1 AND scene=$2`,
		phone, scene).Scan(&recent)
	if err != nil && !errors.Is(err, errNoRows) {
		return err
	}
	if recent {
		return ErrSmsCooldown
	}
	code, err := randDigits(6)
	if err != nil {
		return err
	}
	if _, err = s.pool.Exec(ctx, `INSERT INTO portal_sms_codes(phone, scene, code, expires_at, issued_at)
		VALUES ($1,$2,$3, now() + interval '5 minutes', now())
		ON CONFLICT (phone, scene) DO UPDATE SET code = EXCLUDED.code, expires_at = EXCLUDED.expires_at,
			used = FALSE, issued_at = now()`,
		phone, scene, code); err != nil {
		return err
	}
	if s.sender == nil {
		return nil
	}
	if err := s.sender.Send(ctx, phone, code, scene); err != nil {
		return fmt.Errorf("portal: send sms: %w", err)
	}
	return nil
}

func (s *pgStore) ConsumeSms(ctx context.Context, phone, scene, code string) (bool, error) {
	if code == "" {
		return false, nil
	}
	tag, err := s.pool.Exec(ctx, `UPDATE portal_sms_codes SET used = TRUE
		WHERE phone=$1 AND scene=$2 AND code=$3 AND used=FALSE AND expires_at > now()`,
		phone, scene, code)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// randReader 抽成变量仅为注入故障 reader(测试),生产行为不变。
var randReader = rand.Reader

func randDigits(n int) (string, error) {
	out := make([]byte, n)
	for i := range out {
		v, err := rand.Int(randReader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + v.Int64())
	}
	return string(out), nil
}

// syntheticID 合成客户 ID(负数段,与真实 customers.id 正数段物理隔离)。
// 裁定 2026-08-20: 负数号段 + portal_accounts CHECK(customer_id<>0),防隔离空间合并撞号。
func syntheticID(seq int64) int64 { return -seq }

func (s *pgStore) NextSyntheticCustomerID(ctx context.Context) (int64, error) {
	var last int64
	err := s.pool.QueryRow(ctx,
		`UPDATE portal_seq SET last = last + 1 WHERE kind='CUST' RETURNING last`).Scan(&last)
	if errors.Is(err, errNoRows) {
		if _, err := s.pool.Exec(ctx, `INSERT INTO portal_seq(kind, last) VALUES ('CUST', 1)`); err != nil {
			return 0, err
		}
		return syntheticID(1), nil
	}
	if err != nil {
		return 0, err
	}
	return syntheticID(last), nil
}

func (s *pgStore) UpsertAccount(ctx context.Context, phone, password string, customerID int64) (*Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	a := &Account{Phone: phone, CustomerID: customerID, PasswordHash: string(hash), PasswordUpdatedAt: time.Now()}
	_, err = s.pool.Exec(ctx, `INSERT INTO portal_accounts(phone, customer_id, password_hash, password_updated_at)
		VALUES ($1,$2,$3, now())
		ON CONFLICT (phone) DO UPDATE SET password_hash = EXCLUDED.password_hash, password_updated_at = now()`,
		phone, customerID, a.PasswordHash)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *pgStore) AccountByPhone(ctx context.Context, phone string) (*Account, error) {
	a := &Account{}
	err := s.pool.QueryRow(ctx, `SELECT phone, customer_id, password_hash, password_updated_at
		FROM portal_accounts WHERE phone=$1`, phone).
		Scan(&a.Phone, &a.CustomerID, &a.PasswordHash, &a.PasswordUpdatedAt)
	if errors.Is(err, errNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *pgStore) AccountByCustomer(ctx context.Context, customerID int64) (*Account, error) {
	a := &Account{}
	err := s.pool.QueryRow(ctx, `SELECT phone, customer_id, password_hash, password_updated_at
		FROM portal_accounts WHERE customer_id=$1`, customerID).
		Scan(&a.Phone, &a.CustomerID, &a.PasswordHash, &a.PasswordUpdatedAt)
	if errors.Is(err, errNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (s *pgStore) VerifyPassword(ctx context.Context, phone, password string) (bool, error) {
	a, err := s.AccountByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) == nil, nil
}

func (s *pgStore) RebindPhone(ctx context.Context, customerID int64, newPhone string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE portal_accounts SET phone=$1 WHERE customer_id=$2`, newPhone, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *pgStore) NextNo(ctx context.Context, kind string) (string, error) {
	var last int64
	err := s.pool.QueryRow(ctx, `UPDATE portal_seq SET last = last + 1 WHERE kind=$1 RETURNING last`, kind).Scan(&last)
	if errors.Is(err, errNoRows) {
		return "", fmt.Errorf("portal: unknown seq kind %q", kind)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", kind, last), nil
}
