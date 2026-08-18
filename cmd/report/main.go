// report 服务入口(阶段9):周期自动生成经营分析报告,留痕 PG 后推送 Kafka(boss-report-snapshots)。
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

	var nt report.Notifier = report.NoopNotifier{}
	if len(cfg.Kafka.Brokers) > 0 {
		kn := report.NewKafkaNotifier(cfg.Kafka.Brokers, cfg.Report.PushTopic)
		nt = kn
		defer kn.Close()
	}

	svc := &report.ReportService{
		Ana: analytics.NewPGStore(pool, cfg.Analytics.MaintUnitCost, cfg.Analytics.PortUnitCost),
		St:  report.NewPGStore(pool),
		Nt:  nt,
	}
	log.Printf("report: started period=%s interval=%s pushTopic=%s",
		cfg.Report.Period, cfg.Report.Interval, cfg.Report.PushTopic)
	tick := func() {
		snap, err := svc.Generate(ctx, cfg.Report.Period, time.Now())
		if err != nil {
			log.Printf("report: generate: %v", err)
			return
		}
		if err := svc.Push(ctx, snap); err != nil {
			log.Printf("report: push skipped: %v", err)
		}
		log.Printf("report: period=%s window=%s~%s generated (id=%d)",
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
