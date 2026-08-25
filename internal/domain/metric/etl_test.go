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

func TestNormalizeRunRejectsInvalidLifecycle(t *testing.T) {
	started := time.Now()
	finished := started.Add(-time.Second)
	cases := []ETLJobRun{
		{JobKey: "orders", Status: "UNKNOWN", StartedAt: started},
		{JobKey: "orders", Status: ETLRunning, StartedAt: started, FinishedAt: &started},
		{JobKey: "orders", Status: ETLSuccess, StartedAt: started, FinishedAt: &finished},
	}
	for _, run := range cases {
		if _, err := normalizeRun(run); err == nil {
			t.Errorf("normalizeRun(%+v) accepted invalid lifecycle", run)
		}
	}
}

func TestNormalizeRunCompletesMissingFinishedAt(t *testing.T) {
	run, err := normalizeRun(ETLJobRun{JobKey: "orders", Status: ETLSuccess})
	if err != nil || run.FinishedAt == nil {
		t.Fatalf("run=%+v err=%v", run, err)
	}
}

func ptr(t time.Time) *time.Time { return &t }

// TestFreshnessExcludesDisabledJobs 禁用任务不进新鲜度看板/不产逾期派单。
func TestFreshnessExcludesDisabledJobs(t *testing.T) {
	s := NewMemoryETLStore()
	if _, err := s.UpsertJob(context.Background(), ETLJob{JobKey: "live", Name: "有执行器", LatenessThreshold: 60}); err != nil {
		t.Fatal(err)
	}
	noExec, _ := s.UpsertJob(context.Background(), ETLJob{JobKey: "placeholder", Name: "无执行器", LatenessThreshold: 15})
	noExec.Enabled = false
	if _, err := s.UpsertJob(context.Background(), *noExec); err != nil {
		t.Fatal(err)
	}
	fs, err := s.ListFreshness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.JobKey == "placeholder" {
			t.Fatalf("disabled job leaked into freshness: %+v", fs)
		}
	}
	if len(fs) != 1 || fs[0].JobKey != "live" {
		t.Fatalf("freshness=%+v", fs)
	}
}
