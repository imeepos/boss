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

// ErrInvalidSubject 主体类型非法。
var ErrInvalidSubject = errors.New("apikey: invalid subject type")

// dbtx 是 PGStore 依赖的最小数据库接口。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) (pgx.Row)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 Service 的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造 PGStore。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// listCols 主体名经三表 LEFT JOIN 冗余展示(与迁移 000044 主体模型对齐)。
const listCols = `k.id, k.subject_type, k.subject_ref,
	COALESCE(a.username, w.name, c.name, '') AS subject_name,
	k.name, k.status, k.last_used_at, k.created_at`

const listJoins = `
	FROM api_keys k
	LEFT JOIN accounts a  ON k.subject_type = 'account'  AND a.id = k.subject_ref
	LEFT JOIN workers w   ON k.subject_type = 'worker'   AND w.id = k.subject_ref
	LEFT JOIN customers c ON k.subject_type = 'customer' AND c.id = k.subject_ref`

// Create 为指定主体创建 API key,返回完整密钥(仅在此返回一次)。
// subjectType ∈ {account, worker, customer};subjectRef 为主体表主键。
func (s *PGStore) Create(ctx context.Context, subjectType string, subjectRef, createdBy int64, name string) (*CreateResult, error) {
	if !ValidSubjectType(subjectType) {
		return nil, ErrInvalidSubject
	}
	plain := newKey()
	h := sha256.Sum256([]byte(plain))
	hexHash := hex.EncodeToString(h[:])
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO api_keys (subject_type, subject_ref, name, key_hash, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, subjectType, subjectRef, name, hexHash, createdBy).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("apikey: create: %w", err)
	}
	return &CreateResult{
		APIKey: APIKey{
			ID: id, SubjectType: subjectType, SubjectRef: subjectRef,
			Name: name, KeyPrefix: plain[:8], Status: 1,
		},
		PlainKey: plain,
	}, nil
}

// List 列出所有 API key 元数据(不含明文密钥)。
func (s *PGStore) List(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.Query(ctx, `SELECT `+listCols+listJoins+` ORDER BY k.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("apikey: list: %w", err)
	}
	defer rows.Close()
	out := make([]APIKey, 0)
	for rows.Next() {
		var k APIKey
		var lastUsed, createdAt pgtype.Timestamptz
		if err := rows.Scan(&k.ID, &k.SubjectType, &k.SubjectRef, &k.SubjectName, &k.Name,
			&k.Status, &lastUsed, &createdAt); err != nil {
			return nil, fmt.Errorf("apikey: scan: %w", err)
		}
		if lastUsed.Valid {
			k.LastUsedAt = lastUsed.Time.Format(time.RFC3339)
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

// Lookup 通过密钥哈希查找绑定主体(仅 status=1)。
func (s *PGStore) Lookup(ctx context.Context, keyHash string) (*Subject, error) {
	var subj Subject
	err := s.db.QueryRow(ctx, `
		SELECT subject_type, subject_ref FROM api_keys
		WHERE key_hash = $1 AND status = 1`, keyHash).Scan(&subj.Type, &subj.Ref)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("apikey: lookup: %w", err)
	}
	return &subj, nil
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