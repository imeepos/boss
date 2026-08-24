package app

// ETL 投影执行器:按台账 refresh_cadence 驱动有真实执行器的任务执行真实投影,
// 并写正式运行记录(RUNNING→SUCCESS/FAILED),替代 API 手动回填与扫描循环占位。
// 无真实投影逻辑的任务(order/billing/customer projection 等)保持台账,不伪造记录。

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ymm-001/boss/internal/domain/metric"
)

// etlProjectionFunc 真实投影执行器;返回处理行数与错误。
type etlProjectionFunc func(ctx context.Context) (int64, error)

// etlProjections 台账 jobKey → 真实投影执行器;仅注册有真实投影逻辑的任务。
func etlProjections(a *Application) map[string]etlProjectionFunc {
	m := map[string]etlProjectionFunc{}
	if a.ARClosure != nil {
		m["ar_aging_snapshot"] = func(ctx context.Context) (int64, error) {
			n, err := a.ARClosure.GenerateAgingSnapshots(ctx, time.Now())
			return int64(n), err
		}
	}
	if a.Metric != nil {
		m["metric_quality_scan"] = func(ctx context.Context) (int64, error) {
			v, err := a.Metric.ScanQuality(ctx, "")
			if err != nil {
				return 0, err
			}
			return int64(len(v)), nil
		}
	}
	return m
}

func etlExecutorInterval() time.Duration {
	if raw := os.Getenv("ETL_EXECUTOR_INTERVAL_SECONDS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 60 * time.Second
}

// startETLProjectionLoop 启动 ETL 投影执行器循环,返回 stop(幂等)。
func startETLProjectionLoop(a *Application) func() {
	if a == nil || a.ETL == nil {
		return func() {}
	}
	execs := etlProjections(a)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runETLProjectionsOnce(ctx, a.ETL, execs)
		t := time.NewTicker(etlExecutorInterval())
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runETLProjectionsOnce(ctx, a.ETL, execs)
			}
		}
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
}

// runETLProjectionsOnce 跑一轮:启用且到期的任务,调用真实执行器并写运行记录。
// 无执行器任务跳过(不伪造记录);单任务失败不中断本轮。
func runETLProjectionsOnce(ctx context.Context, etl metric.ETLService, execs map[string]etlProjectionFunc) {
	jobs, err := etl.ListJobs(ctx)
	if err != nil {
		log.Printf("etl executor: list jobs: %v", err)
		return
	}
	for _, j := range jobs {
		if !j.Enabled {
			continue
		}
		fn, ok := execs[j.JobKey]
		if !ok {
			continue // 无真实投影实现,保持台账
		}
		if !etlProjectionDue(j) {
			continue
		}
		started := time.Now()
		if err := etl.RecordRun(ctx, metric.ETLJobRun{JobKey: j.JobKey, StartedAt: started, Status: metric.ETLRunning}); err != nil {
			log.Printf("etl executor %s: record running: %v", j.JobKey, err)
			continue
		}
		rows, runErr := fn(ctx)
		finished := time.Now()
		status := metric.ETLSuccess
		errMsg := ""
		if runErr != nil {
			status = metric.ETLFailed
			errMsg = runErr.Error()
		}
		if err := etl.RecordRun(ctx, metric.ETLJobRun{JobKey: j.JobKey, StartedAt: started, FinishedAt: &finished, Status: status, RowsAffected: rows, Error: errMsg}); err != nil {
			log.Printf("etl executor %s: record finish: %v", j.JobKey, err)
		}
	}
}

// etlProjectionDue 首次(从未运行)立即执行;之后按 refresh_cadence 判断是否到期。
func etlProjectionDue(j metric.ETLJob) bool {
	if j.LastRunAt == nil {
		return true
	}
	return time.Since(*j.LastRunAt) >= etlCadence(j.RefreshCadence)
}

// etlCadence 台账 cadence 文本 → 间隔;未知值保守按 1 小时。
func etlCadence(raw string) time.Duration {
	switch raw {
	case "5MIN":
		return 5 * time.Minute
	case "10MIN":
		return 10 * time.Minute
	case "15MIN":
		return 15 * time.Minute
	case "HOURLY":
		return time.Hour
	case "DAILY":
		return 24 * time.Hour
	}
	return time.Hour
}
