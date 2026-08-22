package quadlink

// Q2 四码冲突 4 小时清零率(000114 时间线):按发现窗口统计冲突事件,
// cleared_within=4h 占比;未清零冲突计入分母(未清零视同未达标)。
// 注意:Reconcile 自动清理(PurgeOrphans)删除的行不参与统计,口径
// 为"进入 CONFLICT 且经人工 ResolveConflict 清零"的事件。

import (
	"context"
	"fmt"
	"strconv"
)

// ClearanceWindowDays 清零率统计窗口(天)。
const ClearanceWindowDays = 7

// ClearanceStats 冲突清零统计(窗口内)。
type ClearanceStats struct {
	Discovered      int64   `json:"discovered"`      // 窗口内发现的冲突事件数
	ClearedWithin4h int64   `json:"clearedWithin4h"` // 其中 4 小时内清零
	StillOpen       int64   `json:"stillOpen"`       // 当前仍在冲突态
	Rate            float64 `json:"rate"`            // ClearedWithin4h/Discovered,无事件=1
}

// ClearanceStats 按 7 天窗口统计冲突清零率(PGStore 窄口,admin 指标用)。
func (s *PGStore) ClearanceStats(ctx context.Context) (*ClearanceStats, error) {
	var st ClearanceStats
	err := s.db.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE cleared_at IS NOT NULL
		                        AND cleared_at - conflict_at <= interval '4 hours'),
		       count(*) FILTER (WHERE status = 'CONFLICT')
		FROM quad_links
		WHERE conflict_at >= now() - ($1 || ' days')::interval`, strconv.Itoa(ClearanceWindowDays)).
		Scan(&st.Discovered, &st.ClearedWithin4h, &st.StillOpen)
	if err != nil {
		return nil, fmt.Errorf("quadlink: clearance stats: %w", err)
	}
	st.Rate = 1
	if st.Discovered > 0 {
		st.Rate = float64(st.ClearedWithin4h) / float64(st.Discovered)
	}
	return &st, nil
}

// ClearanceStatReader 窄口能力(PGStore 实现,handler 侧断言)。
type ClearanceStatReader interface {
	ClearanceStats(ctx context.Context) (*ClearanceStats, error)
}
