package report

// 报告域单测:周期窗口、结论可解释、幂等快照、未知周期拒绝。

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/analytics"
)

type stubAna struct {
	ind   []analytics.Indicator
	rois  []analytics.RegionROI
	heat  []analytics.HeatCell
	maint []analytics.MaintenanceItem
}

func (s stubAna) FiveIndicators(context.Context) ([]analytics.Indicator, []analytics.RegionROI, error) {
	return s.ind, s.rois, nil
}
func (s stubAna) Heatmap(context.Context) ([]analytics.HeatCell, error) { return s.heat, nil }
func (s stubAna) MaintenanceList(context.Context) ([]analytics.MaintenanceItem, error) {
	return s.maint, nil
}

type stubStore struct{ saved []Snapshot }

func (s *stubStore) UpsertSnapshot(_ context.Context, snap *Snapshot) error {
	snap.ID = int64(len(s.saved) + 1)
	s.saved = append(s.saved, *snap)
	return nil
}
func (s *stubStore) LatestSnapshot(_ context.Context, period string) (*Snapshot, error) {
	for _, snap := range s.saved {
		if snap.Period == period {
			return &snap, nil
		}
	}
	return nil, ErrNoSnapshot
}
func (s *stubStore) ListSnapshots(context.Context) ([]Snapshot, error) { return s.saved, nil }
func (s *stubStore) SnapshotByID(_ context.Context, id int64) (*Snapshot, error) {
	for _, snap := range s.saved {
		if snap.ID == id {
			return &snap, nil
		}
	}
	return nil, ErrNoSnapshot
}

func sampleIndicators() []analytics.Indicator {
	return []analytics.Indicator{
		{Key: "portUtilization", Name: "端口利用率", Value: 0.8, Detail: "8/10"},
		{Key: "installConversion", Name: "装机转化率", Value: 0.6, Detail: "6/10"},
		{Key: "maintenanceCostPerUser", Name: "单用户维护成本(元)", Value: 100, Detail: "x"},
		{Key: "assetHealth", Name: "资产健康度评分", Value: 70, Detail: "AVG"},
		{Key: "regionROI", Name: "区域投资回报率", Value: 1.2, Detail: "x"},
	}
}

func TestGenerate(t *testing.T) {
	at := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	ana := stubAna{
		ind:   sampleIndicators(),
		heat:  []analytics.HeatCell{{AddressID: 1, Name: "小区A", Utilization: 0.9}},
		maint: []analytics.MaintenanceItem{{DeviceNo: "OLT-01", Priority: "MUST_REPLACE"}},
	}
	st := &stubStore{}
	r := &ReportService{Ana: ana, St: st}

	snap, err := r.Generate(context.Background(), "daily", at)
	if err != nil {
		t.Fatal(err)
	}
	// 日窗 = [at-24h, at)。
	if !snap.WindowStart.Equal(at.Add(-24*time.Hour)) || !snap.WindowEnd.Equal(at) {
		t.Fatalf("window=%v~%v", snap.WindowStart, snap.WindowEnd)
	}
	var p Payload
	if err := json.Unmarshal(snap.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Indicators) != 5 || len(p.Conclusions) != 3 {
		t.Fatalf("payload=%+v", p)
	}
	if len(p.Conclusions[0]) == 0 || p.Maintenance[0].Priority != "MUST_REPLACE" {
		t.Fatalf("conclusions=%v", p.Conclusions)
	}
	// 高利用率结论应包含 80% 与热点数 1。
	if !contains(p.Conclusions[0], "80%") || !contains(p.Conclusions[0], "1 个") {
		t.Fatalf("conclusion0=%s", p.Conclusions[0])
	}
	// Latest 取回。
	got, err := r.St.LatestSnapshot(context.Background(), "daily")
	if err != nil || got.Period != "daily" {
		t.Fatalf("latest=%+v err=%v", got, err)
	}
}

func TestGenerate_UnknownPeriod(t *testing.T) {
	r := &ReportService{Ana: stubAna{}, St: &stubStore{}}
	if _, err := r.Generate(context.Background(), "yearly", time.Now()); err == nil {
		t.Fatal("want error for unknown period")
	}
}

func TestLatest_None(t *testing.T) {
	r := &ReportService{Ana: stubAna{}, St: &stubStore{}}
	if _, err := r.St.LatestSnapshot(context.Background(), "daily"); !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("err=%v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || index(s, sub) >= 0)
}

func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
