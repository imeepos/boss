package report

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// ReconCounts(每日对账)形状回归:十条 count 全部执行且结果保序。
func TestPGStore_ReconCounts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 10 项检查:仅 quadlink.conflicts 非 0,其余为 0。
	for i, n := range []int64{0, 0, 3, 0, 0, 0, 0, 0, 0, 0} {
		mock.ExpectQuery(`SELECT count\(\*\)`).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(n))
		_ = i
	}

	s := NewPGStore(mock)
	checks, err := s.ReconCounts(context.Background())
	if err != nil {
		t.Fatalf("ReconCounts: %v", err)
	}
	if len(checks) != 10 {
		t.Fatalf("checks=%d, want 10", len(checks))
	}
	wantDomains := []string{"order", "order", "quadlink", "resource", "billing", "gis", "billing", "loy", "gis", "openplat"}
	for i, c := range checks {
		if c.Domain != wantDomains[i] {
			t.Fatalf("checks[%d].Domain=%s, want %s", i, c.Domain, wantDomains[i])
		}
	}
	if checks[2].Count != 3 || checks[2].OK() {
		t.Fatalf("quadlink check=%+v, want count=3 not OK", checks[2])
	}
	if !checks[0].OK() {
		t.Fatalf("order check should be OK: %+v", checks[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// fakeReconStore 同时实现 Store 与 ReconProber 的最小桩。
type fakeReconStore struct {
	checks []ReconCheck
	saved  []*Snapshot
	latest *Snapshot
}

func (f *fakeReconStore) UpsertSnapshot(_ context.Context, s *Snapshot) error {
	f.saved = append(f.saved, s)
	f.latest = s
	return nil
}
func (f *fakeReconStore) LatestSnapshot(_ context.Context, _ string) (*Snapshot, error) {
	if f.latest == nil {
		return nil, ErrNoSnapshot
	}
	return f.latest, nil
}
func (f *fakeReconStore) LatestSnapshots(context.Context, string, int) ([]Snapshot, error) {
	return nil, nil
}
func (f *fakeReconStore) ListSnapshots(context.Context) ([]Snapshot, error) { return nil, nil }
func (f *fakeReconStore) SnapshotByID(context.Context, int64) (*Snapshot, error) {
	return nil, ErrNoSnapshot
}
func (f *fakeReconStore) ReconCounts(context.Context) ([]ReconCheck, error) {
	return f.checks, nil
}

func TestDailyRecon(t *testing.T) {
	at := time.Date(2026, 8, 26, 3, 0, 0, 0, time.Local)
	t.Run("全清零 AllOK 并落当日快照", func(t *testing.T) {
		st := &fakeReconStore{checks: []ReconCheck{
			{Domain: "order", Name: "stuckReserved", Count: 0},
			{Domain: "gis", Name: "usedPortNoOrder", Count: 0},
		}}
		r := &ReportService{St: st}
		snap, p, err := r.DailyRecon(context.Background(), at)
		if err != nil {
			t.Fatalf("DailyRecon: %v", err)
		}
		if !p.AllOK || snap.Period != PeriodReconDaily {
			t.Fatalf("payload=%+v snap=%+v", p, snap)
		}
		if !snap.WindowStart.Equal(time.Date(2026, 8, 26, 0, 0, 0, 0, time.Local)) {
			t.Fatalf("windowStart=%v", snap.WindowStart)
		}
		got, err := r.LatestRecon(context.Background())
		if err != nil || !got.AllOK {
			t.Fatalf("LatestRecon: %+v %v", got, err)
		}
	})
	t.Run("有异常 AllOK=false", func(t *testing.T) {
		st := &fakeReconStore{checks: []ReconCheck{
			{Domain: "quadlink", Name: "conflicts", Count: 2},
		}}
		r := &ReportService{St: st}
		_, p, err := r.DailyRecon(context.Background(), at)
		if err != nil {
			t.Fatalf("DailyRecon: %v", err)
		}
		if p.AllOK {
			t.Fatal("AllOK should be false")
		}
	})
	t.Run("Store 不支持对账", func(t *testing.T) {
		r := &ReportService{St: &fakePatrolStore{}}
		if _, _, err := r.DailyRecon(context.Background(), at); err != ErrReconUnsupported {
			t.Fatalf("err=%v, want ErrReconUnsupported", err)
		}
	})
}
