package report

// 巡检快照定时化:PatrolOrphans 结果落 report_snapshots(period=daily),
// 幂等覆盖同日窗口;GET /db-patrol/latest 读取。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PatrolPayload 巡检快照载荷(report_snapshots.payload)。
type PatrolPayload struct {
	GeneratedAt  time.Time       `json:"generatedAt"`
	Findings     []OrphanFinding `json:"findings"`
	TotalOrphans int             `json:"totalOrphans"`
}

// PatrolSnapshot 跑一次巡检并落当日快照(同窗口幂等覆盖,复用 UpsertSnapshot)。
func (r *ReportService) PatrolSnapshot(ctx context.Context, at time.Time) (*Snapshot, error) {
	findings, err := r.PatrolOrphans(ctx)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, f := range findings {
		total += f.Orphans
	}
	raw, err := json.Marshal(PatrolPayload{GeneratedAt: at, Findings: findings, TotalOrphans: total})
	if err != nil {
		return nil, fmt.Errorf("report: patrol marshal: %w", err)
	}
	start, end, err := windowOf("daily", at)
	if err != nil {
		return nil, err
	}
	snap := &Snapshot{Period: "daily", WindowStart: start, WindowEnd: end, Payload: raw, CreatedAt: at}
	if err := r.St.UpsertSnapshot(ctx, snap); err != nil {
		return nil, fmt.Errorf("report: patrol save: %w", err)
	}
	return snap, nil
}

// LatestPatrol 最新巡检快照(解析为 PatrolPayload);无快照返回 ErrNoSnapshot。
func (r *ReportService) LatestPatrol(ctx context.Context) (*PatrolPayload, error) {
	snap, err := r.St.LatestSnapshot(ctx, "daily")
	if err != nil {
		return nil, err
	}
	var p PatrolPayload
	if err := json.Unmarshal(snap.Payload, &p); err != nil {
		return nil, fmt.Errorf("report: patrol unmarshal: %w", err)
	}
	return &p, nil
}
