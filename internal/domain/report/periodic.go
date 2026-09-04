package report

// 周期报告到期补生成:周/月/季报快照此前无任何生产者,报告中心
// GET /reports/latest?period=weekly|monthly|quarterly 恒 404,前端"查看"
// 恒报"加载失败"。巡检循环每小时调一次 EnsureFresh,到期(最新快照
// window_end 已滑出当前窗口)才真正生成,不放大快照量;daily 由
// PatrolSnapshot 每小时覆盖,不在此列。

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// AutoPeriods 巡检循环代管的报告周期(daily 归 PatrolSnapshot)。
var AutoPeriods = []string{"weekly", "monthly", "quarterly"}

// periodWindow 周期窗口长度(windowOf 与 EnsureFresh 同源,防口径漂移)。
func periodWindow(period string) (time.Duration, error) {
	switch period {
	case "daily":
		return 24 * time.Hour, nil
	case "weekly":
		return 7 * 24 * time.Hour, nil
	case "monthly":
		return 30 * 24 * time.Hour, nil
	case "quarterly":
		return 90 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("report: unknown period %q", period)
	}
}

// EnsureFresh 当前窗口尚无快照时生成一份(Generate 幂等:同窗口覆盖)。
// 到期判定:最新快照 windowEnd 距 now 不足一个周期 → 未到期,跳过。
// 返回是否生成。
func (r *ReportService) EnsureFresh(ctx context.Context, period string, now time.Time) (bool, error) {
	d, err := periodWindow(period)
	if err != nil {
		return false, err
	}
	latest, err := r.St.LatestSnapshot(ctx, period)
	switch {
	case err == nil && now.Sub(latest.WindowEnd) < d:
		return false, nil
	case err != nil && !errors.Is(err, ErrNoSnapshot):
		return false, fmt.Errorf("report: ensure %s latest: %w", period, err)
	}
	if _, err := r.Generate(ctx, period, now); err != nil {
		return false, fmt.Errorf("report: ensure %s generate: %w", period, err)
	}
	return true, nil
}
