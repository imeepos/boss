package metric

import (
	"context"
	"testing"
	"time"
)

func TestMemoryETLStoreRunAndFreshness(t *testing.T) {
	s := NewMemoryETLStore()
	if _, err := s.UpsertJob(context.Background(), ETLJob{JobKey: "orders", Name: "订单投影", LatenessThreshold: 10}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-5 * time.Minute)
	if err := s.RecordRun(context.Background(), ETLJobRun{JobKey: "orders", StartedAt: started, Status: ETLSuccess, RowsAffected: 4}); err != nil {
		t.Fatal(err)
	}
	fs, err := s.ListFreshness(context.Background())
	if err != nil || len(fs) != 1 || fs[0].Status != Fresh {
		t.Fatalf("freshness=%+v err=%v", fs, err)
	}
	runs, err := s.ListRuns(context.Background(), "orders")
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs=%+v err=%v", runs, err)
	}
}
func TestFreshnessStates(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		age  time.Duration
		want string
	}{{5 * time.Minute, Fresh}, {15 * time.Minute, Stale}, {25 * time.Minute, Overdue}} {
		j := ETLJob{LastRunAt: ptr(now.Add(-tc.age)), LatenessThreshold: 10}
		if got := freshness(now, j); got != tc.want {
			t.Errorf("age %v got %s want %s", tc.age, got, tc.want)
		}
	}
}
func ptr(t time.Time) *time.Time { return &t }
