package device

import (
	"context"
	"testing"
	"time"
)

type fakeDevSvc struct {
	DeviceService
	metrics []DeviceMetric
}

func (f *fakeDevSvc) AppendMetric(_ context.Context, m DeviceMetric) (int64, error) {
	f.metrics = append(f.metrics, m)
	return 1, nil
}

type fakeAlarmSvc struct {
	AlarmService
	alarms []Alarm
}

func (f *fakeAlarmSvc) CreateAlarm(_ context.Context, a Alarm) (int64, error) {
	f.alarms = append(f.alarms, a)
	return 1, nil
}

func TestCollectorIngest(t *testing.T) {
	threshold := 5.0
	dev, alm := &fakeDevSvc{}, &fakeAlarmSvc{}
	c := &Collector{Dev: dev, Alarm: alm, PacketLossAlarmPct: &threshold}

	t.Run("正常样本只入库", func(t *testing.T) {
		loss := 0.1
		if err := c.Ingest(context.Background(), Sample{
			ResourceID: 1, PacketLoss: &loss, Status: "ONLINE", CollectedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		if len(dev.metrics) != 1 || len(alm.alarms) != 0 {
			t.Fatalf("m=%d a=%d", len(dev.metrics), len(alm.alarms))
		}
	})

	t.Run("丢包越限 → 实时告警", func(t *testing.T) {
		loss := 9.9
		if err := c.Ingest(context.Background(), Sample{
			ResourceID: 1, PacketLoss: &loss, Status: "FAULT", CollectedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		if len(alm.alarms) != 1 || alm.alarms[0].Level != "CRITICAL" {
			t.Fatalf("alarms=%+v", alm.alarms)
		}
	})
}
