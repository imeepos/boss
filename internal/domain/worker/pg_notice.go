package worker

import (
	"context"
	"fmt"
)

// ListNotices 公告列表(含已下架),发布时间倒序。
func (s *PGStore) ListNotices(ctx context.Context) ([]Notice, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, title, category, active, published_at FROM worker_notices ORDER BY published_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("worker: list notices: %w", err)
	}
	defer rows.Close()
	out := make([]Notice, 0)
	for rows.Next() {
		var n Notice
		if err := rows.Scan(&n.ID, &n.Title, &n.Category, &n.Active, &n.PublishedAt); err != nil {
			return nil, fmt.Errorf("worker: scan notice: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// CreateNotice 发布公告(active 缺省上架),返回自增 id。
func (s *PGStore) CreateNotice(ctx context.Context, n Notice) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_notices(title, category, active) VALUES($1,$2,$3) RETURNING id`,
		n.Title, n.Category, n.Active).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: create notice: %w", err)
	}
	return id, nil
}

// ToggleNotice 公告上架/下架切换;未命中返回 ErrNotFound。
func (s *PGStore) ToggleNotice(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE worker_notices SET active = NOT active WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("worker: toggle notice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
