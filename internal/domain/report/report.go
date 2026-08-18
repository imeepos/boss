package report

// Package report 阶段9:自动经营分析报告(周期快照 + 结论建议,可解释)。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ymm-001/boss/internal/domain/analytics"
)

// Snapshot 一份报告快照(操作留痕,非业务基表实体)。
type Snapshot struct {
	ID          int64     `json:"id"`
	Period      string    `json:"period"`      // daily/weekly/monthly/quarterly
	WindowStart time.Time `json:"windowStart"` // 窗口起点(对齐周期)
	WindowEnd   time.Time `json:"windowEnd"`
	Payload     []byte    `json:"-"` // JSON 全文(指标/ROI/热力/维护/结论)
	CreatedAt   time.Time `json:"createdAt"`
}

// Payload 报告正文结构。
type Payload struct {
	GeneratedAt time.Time                   `json:"generatedAt"`
	Indicators  []analytics.Indicator       `json:"indicators"`
	RegionROI   []analytics.RegionROI       `json:"regionROI"`
	HeatmapTop  []analytics.HeatCell        `json:"heatmapTop"`
	Maintenance []analytics.MaintenanceItem `json:"maintenance"`
	Conclusions []string                    `json:"conclusions"`
}

// Store 快照存储口。
type Store interface {
	UpsertSnapshot(ctx context.Context, s *Snapshot) error
	LatestSnapshot(ctx context.Context, period string) (*Snapshot, error)
	ListSnapshots(ctx context.Context) ([]Snapshot, error)
}

// ReportService 报告服务(依赖分析域,接口隔离)。
type ReportService struct {
	Ana analytics.AnalyticsService
	St  Store
	Nt  Notifier // 推送通道(可空,空则 Push 返回 ErrNoNotifier)
}

// Generate 生成一个周期的报告快照(幂等:同窗口覆盖)。
func (r *ReportService) Generate(ctx context.Context, period string, at time.Time) (*Snapshot, error) {
	start, end, err := windowOf(period, at)
	if err != nil {
		return nil, err
	}
	ind, rois, err := r.Ana.FiveIndicators(ctx)
	if err != nil {
		return nil, fmt.Errorf("report: indicators: %w", err)
	}
	heat, err := r.Ana.Heatmap(ctx)
	if err != nil {
		return nil, fmt.Errorf("report: heatmap: %w", err)
	}
	maint, err := r.Ana.MaintenanceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("report: maintenance: %w", err)
	}
	p := Payload{
		GeneratedAt: at, Indicators: ind, RegionROI: rois,
		HeatmapTop: topHeat(heat, 10), Maintenance: maint, Conclusions: conclude(ind, heat, maint),
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("report: marshal: %w", err)
	}
	snap := &Snapshot{Period: period, WindowStart: start, WindowEnd: end, Payload: raw, CreatedAt: at}
	if err := r.St.UpsertSnapshot(ctx, snap); err != nil {
		return nil, fmt.Errorf("report: save: %w", err)
	}
	return snap, nil
}

// Latest 指定周期最新快照。
func (r *ReportService) Latest(ctx context.Context, period string) (*Snapshot, error) {
	return r.St.LatestSnapshot(ctx, period)
}

// List 快照列表(新→旧)。
func (r *ReportService) List(ctx context.Context) ([]Snapshot, error) {
	return r.St.ListSnapshots(ctx)
}

// windowOf 周期窗口:返回 [窗口起点, 窗口终点)。
func windowOf(period string, at time.Time) (time.Time, time.Time, error) {
	d := map[string]time.Duration{
		"daily": 24 * time.Hour, "weekly": 7 * 24 * time.Hour,
		"monthly": 30 * 24 * time.Hour, "quarterly": 90 * 24 * time.Hour,
	}[period]
	if d == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("report: unknown period %q", period)
	}
	return at.Add(-d), at, nil
}

// topHeat 高利用率前 n(热力图重点区域)。
func topHeat(cells []analytics.HeatCell, n int) []analytics.HeatCell {
	if len(cells) <= n {
		return cells
	}
	return cells[:n]
}

// conclude 自动结论(口径与数据一致,可解释)。
func conclude(ind []analytics.Indicator, heat []analytics.HeatCell, maint []analytics.MaintenanceItem) []string {
	byKey := map[string]analytics.Indicator{}
	for _, i := range ind {
		byKey[i.Key] = i
	}
	out := make([]string, 0, 3)
	if p := byKey["portUtilization"]; p.Value >= 0.7 {
		hot := 0
		for _, c := range heat {
			if c.Utilization >= 0.7 {
				hot++
			}
		}
		out = append(out, fmt.Sprintf("端口利用率 %.0f%%(≥70%%),高负载小区/楼栋 %d 个,建议纳入扩容评估", p.Value*100, hot))
	} else {
		out = append(out, fmt.Sprintf("端口利用率 %.0f%%(低于 70%% 阈值),暂无全域扩容压力", byKey["portUtilization"].Value*100))
	}
	must := 0
	for _, m := range maint {
		if m.Priority == "MUST_REPLACE" {
			must++
		}
	}
	out = append(out, fmt.Sprintf("维护清单 MUST_REPLACE 设备 %d 台,建议按清单顺序优先替换", must))
	out = append(out, fmt.Sprintf("装机转化率 %s,资产健康度 %.1f", byKey["installConversion"].Detail, byKey["assetHealth"].Value))
	return out
}
