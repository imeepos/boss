// collector 服务入口(W7):设备指标轮询入库 + 阈值告警。
// 债务偿还:默认走真实 SNMP 采集源(BOSS_SNMP_TARGETS 配置 code@host),未配置则空轮询(安全默认)。
package main

import (
	"context"
	"log"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// noopPoller 空采集源:无样本(协议适配前的安全默认)。
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

	var poller device.Poller = noopPoller{}
	if len(cfg.Collector.Targets) > 0 {
		poller = buildSNMPPoller(ctx, cfg, resource.NewPGStore(pool))
	}

	store := device.NewPGStore(pool)
	threshold := 5.0
	c := &device.Collector{Dev: store, Alarm: store, PacketLossAlarmPct: &threshold}
	log.Println("collector: started")
	c.RunPoll(ctx, poller, cfg.Collector.Interval)
	log.Println("collector: stopped")
}

// buildSNMPPoller 配置 target("code@host:port") → SNMPPoller,code→资源ID 经资源域解析;未知 code 跳过。
func buildSNMPPoller(ctx context.Context, cfg *config.Config, res resource.ResourceService) *device.SNMPPoller {
	resources, err := res.ListResources(ctx)
	if err != nil {
		log.Printf("collector: list resources: %v", err)
		return &device.SNMPPoller{}
	}
	byCode := make(map[string]int64, len(resources))
	for _, r := range resources {
		byCode[r.Code] = r.ID
	}
	p := &device.SNMPPoller{
		Community: cfg.Collector.Community, OpticalOID: cfg.Collector.OpticalOID,
		PacketLossOID: cfg.Collector.PacketLossOID, StatusOID: cfg.Collector.StatusOID,
	}
	for _, t := range cfg.Collector.Targets {
		code, host, ok := strings.Cut(t, "@")
		if !ok {
			continue
		}
		id, known := byCode[code]
		if !known {
			log.Printf("collector: unknown snmp target code=%s", code)
			continue
		}
		p.Targets = append(p.Targets, device.Target{ResourceID: id, Code: code, Host: host})
	}
	return p
}
