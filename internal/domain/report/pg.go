package report

// PGStore 报告快照存储(report_snapshots,操作留痕;同窗口幂等覆盖)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 最小数据库接口。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 快照存储实现。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// ErrNoSnapshot 无快照。
var ErrNoSnapshot = errors.New("report: no snapshot")

// UpsertSnapshot 写入/覆盖同窗口快照(返回带 ID)。
func (s *PGStore) UpsertSnapshot(ctx context.Context, snap *Snapshot) error {
	return s.db.QueryRow(ctx, `
		INSERT INTO report_snapshots(period, window_start, window_end, payload)
		VALUES($1,$2,$3,$4::jsonb)
		ON CONFLICT (period, window_start) DO UPDATE SET payload = EXCLUDED.payload, created_at = now()
		RETURNING id, created_at`,
		snap.Period, snap.WindowStart, snap.WindowEnd, string(snap.Payload)).
		Scan(&snap.ID, &snap.CreatedAt)
}

// LatestSnapshot 指定周期最新一份。
func (s *PGStore) LatestSnapshot(ctx context.Context, period string) (*Snapshot, error) {
	var snap Snapshot
	err := s.db.QueryRow(ctx, `
		SELECT id, period, window_start, window_end, payload, created_at
		FROM report_snapshots WHERE period = $1
		ORDER BY window_start DESC LIMIT 1`, period).
		Scan(&snap.ID, &snap.Period, &snap.WindowStart, &snap.WindowEnd, &snap.Payload, &snap.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoSnapshot
	}
	if err != nil {
		return nil, fmt.Errorf("report: latest: %w", err)
	}
	return &snap, nil
}

// SnapshotByID 按主键取快照;未命中返回 ErrNoSnapshot。
func (s *PGStore) SnapshotByID(ctx context.Context, id int64) (*Snapshot, error) {
	var snap Snapshot
	err := s.db.QueryRow(ctx, `
		SELECT id, period, window_start, window_end, payload, created_at
		FROM report_snapshots WHERE id = $1`, id).
		Scan(&snap.ID, &snap.Period, &snap.WindowStart, &snap.WindowEnd, &snap.Payload, &snap.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoSnapshot
	}
	if err != nil {
		return nil, fmt.Errorf("report: by id: %w", err)
	}
	return &snap, nil
}

// LatestSnapshots 指定周期最近 limit 份快照(新→旧)。同窗口被 upsert 覆盖,
// 取到的每行 window_start 唯一;limit<=0 返回空切片(不报错)。
func (s *PGStore) LatestSnapshots(ctx context.Context, period string, limit int) ([]Snapshot, error) {
	if limit <= 0 {
		return []Snapshot{}, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, period, window_start, window_end, payload, created_at
		FROM report_snapshots WHERE period = $1
		ORDER BY window_start DESC LIMIT $2`, period, limit)
	if err != nil {
		return nil, fmt.Errorf("report: history: %w", err)
	}
	defer rows.Close()
	out := make([]Snapshot, 0, limit)
	for rows.Next() {
		var snap Snapshot
		if err := rows.Scan(&snap.ID, &snap.Period, &snap.WindowStart, &snap.WindowEnd, &snap.Payload, &snap.CreatedAt); err != nil {
			return nil, fmt.Errorf("report: scan: %w", err)
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

// ListSnapshots 全部快照(新→旧)。
func (s *PGStore) ListSnapshots(ctx context.Context) ([]Snapshot, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, period, window_start, window_end, payload, created_at
		FROM report_snapshots ORDER BY window_start DESC LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("report: list: %w", err)
	}
	defer rows.Close()
	out := make([]Snapshot, 0)
	for rows.Next() {
		var s Snapshot
		if err := rows.Scan(&s.ID, &s.Period, &s.WindowStart, &s.WindowEnd, &s.Payload, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("report: scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
