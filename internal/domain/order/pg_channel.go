package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrChannelDuplicate 渠道 code 重复(channels.code 唯一)。
var ErrChannelDuplicate = errors.New("order: channel code duplicate")

// ListChannels 列出全部渠道目录。
func (s *PGStore) ListChannels(ctx context.Context) ([]Channel, error) {
	rows, err := s.db.Query(ctx, `SELECT id, code, name, status FROM channels ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("order: list channels: %w", err)
	}
	defer rows.Close()
	out := make([]Channel, 0)
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Status); err != nil {
			return nil, fmt.Errorf("order: scan channel: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChannel 按 id 查渠道;未命中返回 ErrOrderNotFound。
func (s *PGStore) GetChannel(ctx context.Context, id int64) (*Channel, error) {
	var c Channel
	err := s.db.QueryRow(ctx, `SELECT id, code, name, status FROM channels WHERE id = $1`, id).
		Scan(&c.ID, &c.Code, &c.Name, &c.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("order: get channel: %w", err)
	}
	return &c, nil
}

// CreateChannel 新增渠道,返回自增 id;code 重复报 ErrChannelDuplicate(40900)。
func (s *PGStore) CreateChannel(ctx context.Context, c Channel) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO channels(code, name, status) VALUES($1,$2,$3) RETURNING id`,
		c.Code, c.Name, c.Status).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return 0, fmt.Errorf("order: channel code %q: %w", c.Code, ErrChannelDuplicate)
	}
	if err != nil {
		return 0, fmt.Errorf("order: create channel: %w", err)
	}
	return id, nil
}
