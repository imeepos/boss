package report

import (
	"context"
	"testing"
	"time"
)

// fakeOSSStore 最小快照存储桩(只记 Upsert 入参)。
type fakeOSSStore struct {
	period string
	saved  int
}

func (f *fakeOSSStore) UpsertSnapshot(_ context.Context, s *Snapshot) error {
	f.period = s.Period
	f.saved++
	return nil
}

func (f *fakeOSSStore) LatestSnapshot(context.Context, string) (*Snapshot, error) {
	return nil, ErrNoSnapshot
}

func (f *fakeOSSStore) LatestSnapshots(context.Context, string, int) ([]Snapshot, error) {
	return nil, nil
}

func (f *fakeOSSStore) ListSnapshots(context.Context) ([]Snapshot, error) { return nil, nil }

func (f *fakeOSSStore) SnapshotByID(context.Context, int64) (*Snapshot, error) {
	return nil, ErrNoSnapshot
}

func TestSaveOSSAudit(t *testing.T) {
	st := &fakeOSSStore{}
	svc := &ReportService{St: st}
	at := time.Date(2026, 9, 6, 4, 0, 0, 0, time.Local)
	payload := map[string]any{"total": float64(6), "category": "ownership"}
	if _, err := svc.SaveOSSAudit(context.Background(), payload, at); err != nil {
		t.Fatalf("save: %v", err)
	}
	if st.period != PeriodOSSAuditDaily {
		t.Fatalf("period = %s, want %s", st.period, PeriodOSSAuditDaily)
	}
	if st.saved != 1 {
		t.Fatalf("saved = %d, want 1", st.saved)
	}
}
