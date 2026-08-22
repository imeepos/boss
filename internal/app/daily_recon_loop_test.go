package app

import (
	"context"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/report"
)

// fakeReconStoreApp report.Store + ReconProber 桩(app 侧)。
type fakeReconStoreApp struct {
	checks     []report.ReconCheck
	saved      []*report.Snapshot
	latestDay  time.Time // 当日已有快照的日期;零值=无快照
}

func (f *fakeReconStoreApp) UpsertSnapshot(_ context.Context, s *report.Snapshot) error {
	f.saved = append(f.saved, s)
	f.latestDay = s.WindowStart
	return nil
}
func (f *fakeReconStoreApp) LatestSnapshot(_ context.Context, period string) (*report.Snapshot, error) {
	if f.latestDay.IsZero() || period != report.PeriodReconDaily {
		return nil, report.ErrNoSnapshot
	}
	return &report.Snapshot{Period: period, WindowStart: f.latestDay}, nil
}
func (f *fakeReconStoreApp) LatestSnapshots(context.Context, string, int) ([]report.Snapshot, error) {
	return nil, nil
}
func (f *fakeReconStoreApp) ListSnapshots(context.Context) ([]report.Snapshot, error) {
	return nil, nil
}
func (f *fakeReconStoreApp) SnapshotByID(context.Context, int64) (*report.Snapshot, error) {
	return nil, report.ErrNoSnapshot
}
func (f *fakeReconStoreApp) ReconCounts(context.Context) ([]report.ReconCheck, error) {
	return f.checks, nil
}

func TestRunDailyReconIfDue(t *testing.T) {
	okChecks := []report.ReconCheck{{Domain: "order", Name: "stuckReserved", Count: 0}}
	badChecks := []report.ReconCheck{
		{Domain: "order", Name: "stuckReserved", Count: 0},
		{Domain: "quadlink", Name: "conflicts", Count: 4},
	}
	t.Run("03:00 前不跑", func(t *testing.T) {
		st := &fakeReconStoreApp{}
		d := reconLoopDeps{rep: &report.ReportService{St: st}, now: func() time.Time {
			return time.Date(2026, 8, 26, 2, 59, 0, 0, time.Local)
		}}
		runDailyReconIfDue(context.Background(), d)
		if len(st.saved) != 0 {
			t.Fatalf("saved=%d, want 0", len(st.saved))
		}
	})
	t.Run("03:00 后异常发 URGENT 待办", func(t *testing.T) {
		st := &fakeReconStoreApp{checks: badChecks}
		n := &fakeNotify{}
		d := reconLoopDeps{rep: &report.ReportService{St: st}, n: n, now: func() time.Time {
			return time.Date(2026, 8, 26, 3, 10, 0, 0, time.Local)
		}}
		runDailyReconIfDue(context.Background(), d)
		if len(st.saved) != 1 {
			t.Fatalf("saved=%d, want 1", len(st.saved))
		}
		if len(n.inputs) != 1 {
			t.Fatalf("notices=%d", len(n.inputs))
		}
		in := n.inputs[0]
		if in.Category != notify.CategoryTodo || in.Level != notify.LevelUrgent ||
			in.RefType != "daily_recon" || in.RefID != "20260826" {
			t.Fatalf("notice=%+v", in)
		}
	})
	t.Run("全清零不发待办", func(t *testing.T) {
		st := &fakeReconStoreApp{checks: okChecks}
		n := &fakeNotify{}
		d := reconLoopDeps{rep: &report.ReportService{St: st}, n: n, now: func() time.Time {
			return time.Date(2026, 8, 26, 3, 10, 0, 0, time.Local)
		}}
		runDailyReconIfDue(context.Background(), d)
		if len(n.inputs) != 0 {
			t.Fatalf("notices=%d, want 0", len(n.inputs))
		}
	})
	t.Run("当日已跑幂等跳过", func(t *testing.T) {
		st := &fakeReconStoreApp{checks: badChecks, latestDay: time.Date(2026, 8, 26, 0, 0, 0, 0, time.Local)}
		n := &fakeNotify{}
		d := reconLoopDeps{rep: &report.ReportService{St: st}, n: n, now: func() time.Time {
			return time.Date(2026, 8, 26, 15, 0, 0, 0, time.Local)
		}}
		runDailyReconIfDue(context.Background(), d)
		if len(st.saved) != 0 || len(n.inputs) != 0 {
			t.Fatalf("saved=%d notices=%d, want 0/0", len(st.saved), len(n.inputs))
		}
	})
}

func TestStartDailyReconLoop(t *testing.T) {
	t.Run("Report 缺失空操作/stop 幂等", func(t *testing.T) {
		startDailyReconLoop(&Application{})()
		stop := startDailyReconLoop(&Application{Report: &report.ReportService{St: &fakeReconStoreApp{}}})
		stop()
		stop()
	})
}
