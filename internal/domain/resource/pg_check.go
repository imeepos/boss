package resource

import (
	"context"
	"fmt"
)

// Check 资源核查(环节2):目标地址是否有空闲端口;options 为空闲端口码列表。
func (s *PGStore) Check(ctx context.Context, addressID int64) (bool, []string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.port_code
		FROM ports p JOIN resources r ON p.resource_id = r.id
		WHERE r.address_id = $1 AND p.status = 'IDLE'
		ORDER BY p.id`, addressID)
	if err != nil {
		return false, nil, fmt.Errorf("resource: check: %w", err)
	}
	defer rows.Close()
	options := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return false, nil, fmt.Errorf("resource: scan port: %w", err)
		}
		options = append(options, code)
	}
	return len(options) > 0, options, rows.Err()
}
