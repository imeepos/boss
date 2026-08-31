package worker

import (
	"context"
	"fmt"
)

// 区域子树匹配(祖先或自身):派单范围从"区域 ID 严格相等"升级为
// "工单区域落在师傅负责区域子树内"。regions 是 L0 共享表,worker 域直查
// 与 onboarding 的区域校验同口径;ltree <@ 为自身含祖先的包含算子。

// MatchedRegionIDs 批量判定工单区域是否落在师傅负责区域子树内。
// 返回映射只含 true 项,调用方以 `matched[id]` 取用(零值即 false)。
func (s *PGStore) MatchedRegionIDs(ctx context.Context, workerRegionID int64, ticketRegionIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(ticketRegionIDs))
	if workerRegionID == 0 {
		for _, id := range ticketRegionIDs {
			out[id] = true // 未设区域=不限区域
		}
		return out, nil
	}
	pending := make([]int64, 0, len(ticketRegionIDs))
	for _, id := range ticketRegionIDs {
		if id == 0 {
			out[id] = true // 无区域工单放行
			continue
		}
		pending = append(pending, id)
	}
	if len(pending) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT id FROM regions
		WHERE id = ANY($1)
		  AND path <@ (SELECT path FROM regions WHERE id = $2)`, pending, workerRegionID)
	if err != nil {
		return nil, fmt.Errorf("worker: matched region ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("worker: scan matched region: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}
