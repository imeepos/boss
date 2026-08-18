package userdata

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore Service 接口的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// listMaps 执行查询并把每行折叠为「列名→值」map(列名即 JSON 字段)。
func (s *PGStore) listMaps(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("userdata: query: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("userdata: row values: %w", err)
		}
		m := make(map[string]any, len(vals))
		for i, name := range rows.FieldDescriptions() {
			m[name.Name] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// getMap 单行查询;未命中返回 ErrNotFound。
func (s *PGStore) getMap(ctx context.Context, sql string, args ...any) (map[string]any, error) {
	list, err := s.listMaps(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	return list[0], nil
}

// execAffected 执行写语句并校验命中行数;0 行返回 ErrNotFound。
func (s *PGStore) execAffected(ctx context.Context, op string, sql string, args ...any) error {
	tag, err := s.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("userdata: %s: %w", op, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// insertReturning 执行插入并返回自增 id。
func (s *PGStore) insertReturning(ctx context.Context, op, sql string, args ...any) (int64, error) {
	var id int64
	if err := s.db.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
		return 0, fmt.Errorf("userdata: %s: %w", op, err)
	}
	return id, nil
}
