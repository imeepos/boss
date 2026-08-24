package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/metric"
)

func TestETLProjectionDue(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		j    metric.ETLJob
		want bool
	}{
		{"never run", metric.ETLJob{}, true},
		{"5min cadence stale", metric.ETLJob{LastRunAt: ptrTime(now.Add(-6 * time.Minute)), RefreshCadence: "5MIN"}, true},
		{"5min cadence fresh", metric.ETLJob{LastRunAt: ptrTime(now.Add(-2 * time.Minute)), RefreshCadence: "5MIN"}, false},
		{"hourly stale", metric.ETLJob{LastRunAt: ptrTime(now.Add(-2 * time.Hour)), RefreshCadence: "HOURLY"}, true},
		{"hourly fresh", metric.ETLJob{LastRunAt: ptrTime(now.Add(-30 * time.Minute)), RefreshCadence: "HOURLY"}, false},
		{"unknown cadence stale", metric.ETLJob{LastRunAt: ptrTime(now.Add(-2 * time.Hour)), RefreshCadence: "WEIRDLY"}, true},
	}
	for _, tc := range cases {
		if got := etlProjectionDue(tc.j); got != tc.want {
			t.Errorf("%s: due=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestRunETLProjectionsOnce(t *testing.T) {
	store := metric.NewMemoryETLStore()
	ctx := context.Background()
	// 注册四个任务:两个有真实执行器,两个无执行器(保持台账不伪造)。
	for _, j := range []metric.ETLJob{
		{JobKey: "ar_aging_snapshot", Name: "AR账龄快照", Enabled: true},
		{JobKey: "metric_quality_scan", Name: "指标质量扫描", Enabled: true},
		{JobKey: "order_projection", Name: "订单投影", Enabled: true},
		{JobKey: "disabled_job", Name: "停用任务", Enabled: false},
	} {
		if _, err := store.UpsertJob(ctx, j); err != nil {
			t.Fatal(err)
		}
	}
	execs := map[string]etlProjectionFunc{
		"ar_aging_snapshot": func(context.Context) (int64, error) { return 42, nil },
		"metric_quality_scan": func(context.Context) (int64, error) {
			return 0, errors.New("scan failed")
		},
	}

	runETLProjectionsOnce(ctx, store, execs)

	runs, err := store.ListRuns(ctx, "ar_aging_snapshot")
	if err != nil || len(runs) != 2 {
		t.Fatalf("ar runs=%+v err=%v want RUNNING+SUCCESS", runs, err)
	}
	if runs[0].Status != metric.ETLRunning || runs[1].Status != metric.ETLSuccess || runs[1].RowsAffected != 42 {
		t.Errorf("ar runs=%+v want RUNNING then SUCCESS rows=42", runs)
	}

	runs, err = store.ListRuns(ctx, "metric_quality_scan")
	if err != nil || len(runs) != 2 || runs[1].Status != metric.ETLFailed || runs[1].Error == "" {
		t.Fatalf("quality runs=%+v err=%v want RUNNING then FAILED", runs, err)
	}

	runs, err = store.ListRuns(ctx, "order_projection")
	if err != nil || len(runs) != 0 {
		t.Fatalf("order runs=%+v err=%v want none (no executor)", runs, err)
	}

	// 台账 last_status 回写:真实执行的任务为 SUCCESS/FAILED。
	if j, _ := store.GetJob(ctx, "ar_aging_snapshot"); j.LastStatus != metric.ETLSuccess {
		t.Errorf("ar job last_status=%s want SUCCESS", j.LastStatus)
	}
	if j, _ := store.GetJob(ctx, "disabled_job"); j.LastStatus == metric.ETLSuccess {
		t.Errorf("disabled job must not run")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
