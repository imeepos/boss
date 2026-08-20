package report

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// fakePatrolStore 同时实现 Store 与 OrphanPatroller(巡检快照所需最小能力)。
type fakePatrolStore struct {
	Store
	saved    *Snapshot
	calls    int
	findings []OrphanFinding
}

func (f *fakePatrolStore) UpsertSnapshot(_ context.Context, snap *Snapshot) error {
	f.calls++
	f.saved = snap
	return nil
}

func (f *fakePatrolStore) LatestSnapshot(_ context.Context, period string) (*Snapshot, error) {
	if f.saved == nil || f.saved.Period != period {
		return nil, ErrNoSnapshot
	}
	return f.saved, nil
}

func (f *fakePatrolStore) PatrolOrphans(context.Context) ([]OrphanFinding, error) {
	return f.findings, nil
}

// 巡检快照:落库幂等(同窗口覆盖)+载荷结构与汇总正确。
func TestPatrolSnapshot(t *testing.T) {
	at := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	st := &fakePatrolStore{findings: []OrphanFinding{
		{Check: "orders.customer_id -> customers", Orphans: 2, SampleIDs: []int64{7, 8}},
		{Check: "cdrs.loid -> lo_accounts", Orphans: 0},
	}}
	r := &ReportService{St: st}

	snap, err := r.PatrolSnapshot(context.Background(), at)
	if err != nil {
		t.Fatalf("PatrolSnapshot: %v", err)
	}
	if snap.Period != "daily" {
		t.Fatalf("period=%s", snap.Period)
	}
	var p PatrolPayload
	if err := json.Unmarshal(snap.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.TotalOrphans != 2 || len(p.Findings) != 2 {
		t.Fatalf("payload=%+v", p)
	}
	if st.calls != 1 {
		t.Fatalf("calls=%d", st.calls)
	}

	// 同窗口重跑覆盖。
	if _, err := r.PatrolSnapshot(context.Background(), at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if st.calls != 2 {
		t.Fatalf("calls=%d, want 2", st.calls)
	}

	got, err := r.LatestPatrol(context.Background())
	if err != nil {
		t.Fatalf("LatestPatrol: %v", err)
	}
	if got.TotalOrphans != 2 {
		t.Fatalf("got=%+v", got)
	}
}

func TestLatestPatrol_NoSnapshot(t *testing.T) {
	r := &ReportService{St: &fakePatrolStore{}}
	if _, err := r.LatestPatrol(context.Background()); !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("err=%v, want ErrNoSnapshot", err)
	}
}
