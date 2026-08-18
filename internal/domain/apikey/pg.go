package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound 记录不存在或已停用。
var ErrNotFound = errors.New("apikey: not found")

// dbtx 是 PGStore 依赖的最小数据库接口。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 Service 的 PostgreSQL 实现。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

const keyCols = `k.id, k.account_id, a.real_name, k.name, k.status, k.last_used_at, k.expires_at, k.created_at`

// Create 为指定账号创建 API key,返回完整密钥(仅在此返回一次)。
func (s *PGStore) Create(ctx context.Context, accountID, createdBy int64, name string) (*CreateResult, error) {
	plain := newKey()
	h := sha256.Sum256([]byte(plain))
	hexHash := hex.EncodeToString(h[:])
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO api_keys (account_id, name, key_hash, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, accountID, name, hexHash, createdBy).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("apikey: create: %w", err)
	}
	return &CreateResult{
		APIKey: APIKey{
			ID:        id,
			AccountID: accountID,
			Name:      name,
			KeyPrefix: plain[:8],
			Status:    1,
		},
		PlainKey: plain,
	}, nil
}

// List 列出所有 API key 元数据。
func (s *PGStore) List(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+keyCols+`
		FROM api_keys k
		JOIN accounts a ON a.id = k.account_id
		ORDER BY k.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("apikey: list: %w", err)
	}
	defer rows.Close()
	out := make([]APIKey, 0)
	for rows.Next() {
		var k APIKey
		var lastUsed, expires, createdAt pgtype.Timestamptz
		if err := rows.Scan(&k.ID, &k.AccountID, &k.AccountName, &k.Name,
			&k.Status, &lastUsed, &expires, &createdAt); err != nil {
			return nil, fmt.Errorf("apikey: scan: %w", err)
		}
		if lastUsed.Valid {
			k.LastUsedAt = lastUsed.Time.Format(time.RFC3339)
		}
		if expires.Valid {
			k.ExpiresAt = expires.Time.Format(time.RFC3339)
		}
		if createdAt.Valid {
			k.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// Revoke 停用 API key(软删除)。
func (s *PGStore) Revoke(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE api_keys SET status=0 WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("apikey: revoke: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Lookup 通过密钥哈希查找绑定的账号ID(仅 status=1)。
func (s *PGStore) Lookup(ctx context.Context, keyHash string) (int64, error) {
	var accountID int64
	err := s.db.QueryRow(ctx, `
		SELECT account_id FROM api_keys
		WHERE key_hash = $1 AND status = 1`, keyHash).Scan(&accountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("apikey: lookup: %w", err)
	}
	return accountID, nil
}

// Touch 更新 last_used_at(尽力而为,不返回错误)。
func (s *PGStore) Touch(ctx context.Context, keyHash string) {
	_, _ = s.db.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE key_hash = $1`, keyHash)
}

// newKey 生成格式为 boss_<32hex> 的随机密钥。
func newKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("apikey: rand read: %w", err))
	}
	return "boss_" + hex.EncodeToString(b)
}