package report

// EnsureFresh 单测:无快照必补、未到期跳过、过期重生成、未知周期拒绝。

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnsureFresh_NoSnapshot(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	st := &stubStore{}
	r := &ReportService{Ana: stubAna{ind: sampleIndicators()}, St: st}
	ok, err := r.EnsureFresh(context.Background(), "weekly", at)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(st.saved) != 1 || st.saved[0].Period != "weekly" {
		t.Fatalf("saved=%+v", st.saved)
	}
	// 窗口 = [at-7d, at)。
	if !st.saved[0].WindowStart.Equal(at.Add(-7 * 24 * time.Hour)) {
		t.Fatalf("windowStart=%v", st.saved[0].WindowStart)
	}
}

func TestEnsureFresh_SkipWhenFresh(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	st := &stubStore{}
	r := &ReportService{Ana: stubAna{ind: sampleIndicators()}, St: st}
	if _, err := r.Generate(context.Background(), "weekly", at); err != nil {
		t.Fatal(err)
	}
	// 同窗口内再跑:未到期,不落新快照。
	ok, err := r.EnsureFresh(context.Background(), "weekly", at.Add(time.Hour))
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(st.saved) != 1 {
		t.Fatalf("want 1 snapshot, got %d", len(st.saved))
	}
}

func TestEnsureFresh_RegenerateWhenStale(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	st := &stubStore{}
	r := &ReportService{Ana: stubAna{ind: sampleIndicators()}, St: st}
	if _, err := r.Generate(context.Background(), "weekly", at); err != nil {
		t.Fatal(err)
	}
	// 8 天后:窗口滑出,应重生成(月/季同理,周期长度差异由 periodWindow 决定)。
	ok, err := r.EnsureFresh(context.Background(), "weekly", at.Add(8*24*time.Hour))
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(st.saved) != 2 {
		t.Fatalf("want 2 snapshots, got %d", len(st.saved))
	}
}

func TestEnsureFresh_UnknownPeriod(t *testing.T) {
	r := &ReportService{Ana: stubAna{}, St: &stubStore{}}
	if _, err := r.EnsureFresh(context.Background(), "yearly", time.Now()); err == nil {
		t.Fatal("want error for unknown period")
	}
}

func TestWindowOf_AllPeriods(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	want := map[string]time.Duration{
		"daily": 24 * time.Hour, "weekly": 7 * 24 * time.Hour,
		"monthly": 30 * 24 * time.Hour, "quarterly": 90 * 24 * time.Hour,
	}
	for p, d := range want {
		start, end, err := windowOf(p, at)
		if err != nil || !start.Equal(at.Add(-d)) || !end.Equal(at) {
			t.Fatalf("%s start=%v end=%v err=%v", p, start, end, err)
		}
	}
}

func TestEnsureFresh_StoreErrorPropagates(t *testing.T) {
	r := &ReportService{Ana: stubAna{}, St: &errStore{}}
	if _, err := r.EnsureFresh(context.Background(), "weekly", time.Now()); !errors.Is(err, errBoom) {
		t.Fatalf("err=%v", err)
	}
}

var errBoom = errors.New("boom")

type errStore struct{ stubStore }

func (e *errStore) LatestSnapshot(context.Context, string) (*Snapshot, error) {
	return nil, errBoom
}
