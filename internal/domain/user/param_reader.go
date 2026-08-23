package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetParam 读取单个业务参数，供其他域按 key 热读取。
func (s *PGStore) GetParam(ctx context.Context, key string) (string, error) {
	var raw string
	if err := s.db.QueryRow(ctx, `SELECT value::text FROM biz_params WHERE key=$1`, key).Scan(&raw); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("user: get param: %w", err)
	}
	var value string
	if json.Unmarshal([]byte(raw), &value) == nil {
		return value, nil
	}
	return raw, nil
}
