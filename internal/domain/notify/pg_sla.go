package notify

// Q2 P1 待办时限(000115):超时未办就地升级 URGENT + 按时办结率统计。

import (
	"context"
	"fmt"
)

// SLAWindowDays 时限统计窗口(天)。
const SLAWindowDays = 7

// SLAStats 窗口内带时限待办的办结统计。
type SLAStats struct {
	WithDeadline   int64   `json:"withDeadline"`   // 窗口内带时限的待办数
	ResolvedOnTime int64   `json:"resolvedOnTime"` // 其中时限内办结
	ResolvedLate   int64   `json:"resolvedLate"`   // 办结但超时
	OverdueOpen    int64   `json:"overdueOpen"`    // 当前超时未办
	OnTimeRate     float64 `json:"onTimeRate"`     // OnTime/(OnTime+Late+OverdueOpen);无样本=1
}

// EscalateOverdue 超时未办待办就地升级 URGENT(巡检循环每小时调;
// 只动 level 不改 due_at,幂等),返回本轮升级条数。窄口能力。
func (s *PGStore) EscalateOverdue(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE admin_notifications SET level = 'URGENT'
		WHERE category = 'todo' AND NOT resolved
		  AND due_at IS NOT NULL AND due_at < now() AND level <> 'URGENT'`)
	if err != nil {
		return 0, fmt.Errorf("notify: escalate overdue: %w", err)
	}
	return tag.RowsAffected(), nil
}

// OverdueEscalator 窄口断言用(巡检循环)。
type OverdueEscalator interface {
	EscalateOverdue(ctx context.Context) (int64, error)
}

// SLAStats 窗口内时限达标统计(窄口能力,admin 指标用)。
func (s *PGStore) SLAStats(ctx context.Context, windowDays int) (*SLAStats, error) {
	if windowDays <= 0 {
		windowDays = SLAWindowDays
	}
	var st SLAStats
	err := s.db.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE resolved AND resolved_at <= due_at),
		       count(*) FILTER (WHERE resolved AND resolved_at > due_at),
		       count(*) FILTER (WHERE NOT resolved AND due_at < now())
		FROM admin_notifications
		WHERE category = 'todo' AND due_at IS NOT NULL
		  AND created_at >= now() - ($1 || ' days')::interval`,
		fmt.Sprint(windowDays)).
		Scan(&st.WithDeadline, &st.ResolvedOnTime, &st.ResolvedLate, &st.OverdueOpen)
	if err != nil {
		return nil, fmt.Errorf("notify: sla stats: %w", err)
	}
	denom := st.ResolvedOnTime + st.ResolvedLate + st.OverdueOpen
	st.OnTimeRate = 1
	if denom > 0 {
		st.OnTimeRate = float64(st.ResolvedOnTime) / float64(denom)
	}
	return &st, nil
}

// SLAStatReader 窄口断言用(handler 侧)。
type SLAStatReader interface {
	SLAStats(ctx context.Context, windowDays int) (*SLAStats, error)
}
