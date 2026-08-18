// report 服务入口(阶段9):周期自动生成经营分析报告并留痕推送。
// 生成:ReportService.Generate(五大指标/区域 ROI/热力图/维护清单/结论) → report_snapshots;
// 推送:当前实现为日志投递(Kafka/邮件通道属部署层配置,后续接入)。
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/ymm-001/boss/internal/domain/analytics"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("report: open database: %v", err)
	}
	defer pool.Close()

	svc := &report.ReportService{
		Ana: analytics.NewPGStore(pool, cfg.Analytics.MaintUnitCost, cfg.Analytics.PortUnitCost),
		St:  report.NewPGStore(pool),
	}
	log.Printf("report: started period=%s interval=%s", cfg.Report.Period, cfg.Report.Interval)
	tick := func() {
		snap, err := svc.Generate(ctx, cfg.Report.Period, time.Now())
		if err != nil {
			log.Printf("report: generate: %v", err)
			return
		}
		log.Printf("report: period=%s window=%s~%s generated (id=%d), 推送:日志通道",
			snap.Period, snap.WindowStart.Format(time.RFC3339), snap.WindowEnd.Format(time.RFC3339), snap.ID)
	}
	tick()
	t := time.NewTicker(cfg.Report.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("report: stopped")
			return
		case <-t.C:
			tick()
		}
	}
}
