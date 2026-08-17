// collector 服务入口(W7):设备指标轮询入库 + 阈值告警。
// Poller 当前为空桩;SNMP(gosnmp)接入时替换装配即可,Ingest/告警链路不变。
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// noopPoller 桩采集源:无样本(协议适配前的安全默认)。
type noopPoller struct{}

func (noopPoller) Poll(context.Context) ([]device.Sample, error) { return nil, nil }

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("collector: open database: %v", err)
	}
	defer pool.Close()

	store := device.NewPGStore(pool)
	threshold := 5.0
	c := &device.Collector{Dev: store, Alarm: store, PacketLossAlarmPct: &threshold}
	log.Println("collector: started")
	c.RunPoll(ctx, noopPoller{}, 30*time.Second)
	log.Println("collector: stopped")
}
