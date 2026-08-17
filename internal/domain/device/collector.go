package device

// W7 collector:指标轮询入库 + 阈值告警(丢包率/光功率越限即触发,实时告警验收点)。

import (
	"context"
	"fmt"
	"time"
)

// Sample 一次采集样本(来源:SNMP get / Trap 变量绑定)。
type Sample struct {
	ResourceID   int64
	OpticalPower *float64
	PacketLoss   *float64
	Status       string // ONLINE/OFFLINE/FAULT
	CollectedAt  time.Time
}

// Poller 采集源口(SNMP 轮询实现;测试/演示用桩)。
type Poller interface {
	// Poll 拉一轮全量样本。
	Poll(ctx context.Context) ([]Sample, error)
}

// Collector 采集器:样本入库(AppendMetric) + 越限告警(CreateAlarm)。
type Collector struct {
	Dev   DeviceService
	Alarm AlarmService
	// PacketLossAlarmPct 丢包率告警阈值(%);nil=不告警。
	PacketLossAlarmPct *float64
}

// Ingest 样本入库并按阈值产生告警。
func (c *Collector) Ingest(ctx context.Context, s Sample) error {
	if _, err := c.Dev.AppendMetric(ctx, DeviceMetric{
		ResourceID: s.ResourceID, OpticalPower: s.OpticalPower, PacketLoss: s.PacketLoss,
		Status: s.Status, CollectedAt: s.CollectedAt,
	}); err != nil {
		return err
	}
	if c.PacketLossAlarmPct != nil && s.PacketLoss != nil && *s.PacketLoss >= *c.PacketLossAlarmPct {
		no := fmt.Sprintf("ALM-%d", time.Now().UnixNano())
		if _, err := c.Alarm.CreateAlarm(ctx, Alarm{
			AlarmNo: no, ResourceID: s.ResourceID, Level: "CRITICAL", Source: "device",
			Content: "packet loss threshold exceeded", Status: "OPEN", CreatedAt: time.Now(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// RunPoll 轮询采集直至 ctx 取消。
func (c *Collector) RunPoll(ctx context.Context, p Poller, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			samples, err := p.Poll(ctx)
			if err != nil {
				continue
			}
			for _, s := range samples {
				_ = c.Ingest(ctx, s)
			}
		}
	}
}
