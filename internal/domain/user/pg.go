package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGStore 依赖的最小数据库接口:*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 Service 接口的 PostgreSQL 实现(阶段1:组织/账号/数据范围)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// ListLegalEntities 列出全部子公司/法人。
func (s *PGStore) ListLegalEntities(ctx context.Context) ([]LegalEntity, error) {
	rows, err := s.db.Query(ctx, `SELECT id, code, name FROM legal_entities ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("user: list legal_entities: %w", err)
	}
	defer rows.Close()
	out := make([]LegalEntity, 0)
	for rows.Next() {
		var e LegalEntity
		if err := rows.Scan(&e.ID, &e.Code, &e.Name); err != nil {
			return nil, fmt.Errorf("user: scan legal_entity: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListRegions 列出经营区域;parentPath 为空返回全部,否则返回该子树(ltree 前缀)。
func (s *PGStore) ListRegions(ctx context.Context, parentPath string) ([]Region, error) {
	query := `SELECT id, path, level, name FROM regions ORDER BY path`
	args := []any{}
	if parentPath != "" {
		query = `SELECT id, path, level, name FROM regions WHERE path <@ $1::ltree ORDER BY path`
		args = append(args, parentPath)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("user: list regions: %w", err)
	}
	defer rows.Close()
	out := make([]Region, 0)
	for rows.Next() {
		var r Region
		if err := rows.Scan(&r.ID, &r.Path, &r.Level, &r.Name); err != nil {
			return nil, fmt.Errorf("user: scan region: %w", err)
		}
		r.Parent = parentOf(r.Path)
		out = append(out, r)
	}
	return out, rows.Err()
}

// parentOf 由 ltree 物化路径反查父路径(subpath(path,0,-1) 的应用层等价实现)。
func parentOf(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}
