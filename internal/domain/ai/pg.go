// ai 配置存取:biz_params 单键读写,apiKey/apiUrl/model 三键即全部状态,天然热更。
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 最小数据库接口:*pgxpool.Pool 天然满足,单测可注入 mock。
type dbtx interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore biz_params 读写的 PG 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造配置存储;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// getParam 读单键(value 为去引号后的 JSONB 标量);不存在返回空串。
func (s *PGStore) getParam(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRow(ctx,
		`SELECT value::text FROM biz_params WHERE key = $1`, key).Scan(&v)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("ai: get param %s: %w", key, err)
	}
	return unquoteJSON(v), nil
}

// setParam 写单键(不存在则插入);value 按 JSON 字符串存储以满足 JSONB 约束。
func (s *PGStore) setParam(ctx context.Context, key, value string, updatedBy int64) error {
	b, err := json.Marshal(value)
	if err != nil {
		return ErrInvalidInput
	}
	if _, err := s.db.Exec(ctx, `
		INSERT INTO biz_params(key, value, updated_by, updated_at)
		VALUES($1, $2::jsonb, NULLIF($3,0), now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		key, string(b), updatedBy); err != nil {
		return fmt.Errorf("ai: set param %s: %w", key, err)
	}
	return nil
}

// unquoteJSON JSONB 文本转标量:JSON 字符串去引号,其余原样。
func unquoteJSON(s string) string {
	var str string
	if err := json.Unmarshal([]byte(s), &str); err == nil {
		return str
	}
	return s
}
