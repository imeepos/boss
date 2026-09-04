package monthly

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbtx 最小数据库接口(*pgxpool.Pool / pgxmock 均满足)。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 月度填报域 PG 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// ListRegions 区域白名单(名称字典序;active 标记随行)。
func (s *PGStore) ListRegions(ctx context.Context) ([]Region, error) {
	rows, err := s.db.Query(ctx, `SELECT region, active, updated_at FROM monthly_regions ORDER BY region`)
	if err != nil {
		return nil, fmt.Errorf("monthly: list regions: %w", err)
	}
	defer rows.Close()
	out := make([]Region, 0, 64)
	for rows.Next() {
		var r Region
		var ts pgtype.Timestamptz
		if err := rows.Scan(&r.Region, &r.Active, &ts); err != nil {
			return nil, fmt.Errorf("monthly: scan region: %w", err)
		}
		if ts.Valid {
			r.UpdatedAt = ts.Time.Format(time.RFC3339)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
