package report

// Q2 每日数据对账:订单/四码/资源/账务/GIS 五域只读检查,结果落
// report_snapshots(period=recon-daily,同日幂等覆盖),异常由 app 层
// 发 URGENT 待办(P1 责任队列入口)。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// PeriodReconDaily 每日对账快照周期键(与报告 daily 区分)。
const PeriodReconDaily = "recon-daily"

// ReconCheck 单域检查结果;Count=异常条数,0 即 OK。
type ReconCheck struct {
	Domain string `json:"domain"` // order/quadlink/resource/billing/gis
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Count  int64  `json:"count"`
}

// OK 异常清零判定。
func (c ReconCheck) OK() bool { return c.Count == 0 }

// ReconPayload 每日对账快照正文。
type ReconPayload struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Checks      []ReconCheck `json:"checks"`
	AllOK       bool         `json:"allOK"`
}

// ReconProber Store 可选能力:执行五域只读计数检查。
type ReconProber interface {
	ReconCounts(ctx context.Context) ([]ReconCheck, error)
}

// ErrReconUnsupported Store 不支持对账(如测试 fake)。
var ErrReconUnsupported = fmt.Errorf("report: store does not support recon")

// DailyRecon 跑一轮五域对账并落当日快照(同日幂等覆盖),返回正文。
// 单域检查失败即整轮报错(只读计数失败=连接问题,与巡检同语义)。
func (r *ReportService) DailyRecon(ctx context.Context, at time.Time) (*Snapshot, *ReconPayload, error) {
	p, ok := r.St.(ReconProber)
	if !ok {
		return nil, nil, ErrReconUnsupported
	}
	checks, err := p.ReconCounts(ctx)
	if err != nil {
		return nil, nil, err
	}
	all := true
	for _, c := range checks {
		if !c.OK() {
			all = false
		}
	}
	payload := &ReconPayload{GeneratedAt: at, Checks: checks, AllOK: all}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("report: recon marshal: %w", err)
	}
	start := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	snap := &Snapshot{
		Period: PeriodReconDaily, WindowStart: start, WindowEnd: start.Add(24 * time.Hour),
		Payload: raw, CreatedAt: at,
	}
	if err := r.St.UpsertSnapshot(ctx, snap); err != nil {
		return nil, nil, fmt.Errorf("report: recon save: %w", err)
	}
	return snap, payload, nil
}

// LatestRecon 最新每日对账快照(解析为 ReconPayload);无快照返回 ErrNoSnapshot。
func (r *ReportService) LatestRecon(ctx context.Context) (*ReconPayload, error) {
	snap, err := r.St.LatestSnapshot(ctx, PeriodReconDaily)
	if err != nil {
		return nil, err
	}
	var p ReconPayload
	if err := json.Unmarshal(snap.Payload, &p); err != nil {
		return nil, fmt.Errorf("report: recon unmarshal: %w", err)
	}
	return &p, nil
}
