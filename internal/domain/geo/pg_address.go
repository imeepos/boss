package geo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetAddress 按 id 查地址(path/name/level);未命中返回 nil。
func (s *PGStore) GetAddress(ctx context.Context, id int64) (*AddressInfo, error) {
	var a AddressInfo
	err := s.db.QueryRow(ctx,
		`SELECT id, path::text, name, level FROM addresses WHERE id = $1`, id,
	).Scan(&a.ID, &a.Path, &a.Name, &a.Level)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}
