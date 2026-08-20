package app

// 孤儿巡检定时循环(db-design-review D7 定时化):每小时整点跑一次
// Report.PatrolSnapshot 落 report_snapshots;失败仅记日志不重试不中断。
// 随 Application 生命周期启停(Close 时 stop)。

import (
	"context"
	"log"
	"time"
)

// patrolInterval 巡检周期;整点对齐只是观感,错过由下一轮补上。
const patrolInterval = time.Hour

// startPatrolLoop 启动后台巡检循环,返回 stop(幂等)。
// 首轮立即执行一次(启动即有当日基线),此后按周期。
func startPatrolLoop(a *Application) (stop func()) {
	if a.Report == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runPatrolOnce(ctx, a)
		t := time.NewTicker(patrolInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runPatrolOnce(ctx, a)
			}
		}
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second): // 不阻塞退出
		}
	}
}

func runPatrolOnce(ctx context.Context, a *Application) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if _, err := a.Report.PatrolSnapshot(cctx, time.Now()); err != nil {
		log.Printf("patrol loop: %v", err)
	}
}
