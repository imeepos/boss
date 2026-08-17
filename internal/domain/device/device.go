package device

import (
	"context"
	"time"
)

// DeviceMetric 设备监控指标(OLT 光功率/丢包率快照)。
type DeviceMetric struct {
	ID           int64
	ResourceID   int64
	OpticalPower *float64 // nil=离线无数据
	PacketLoss   *float64
	Status       string // ONLINE/OFFLINE/FAULT
	CollectedAt  time.Time
}

// DeviceMaintenance 设备健康观察(师傅端主动运维清单)。
type DeviceMaintenance struct {
	ID          int64
	DeviceNo    string
	DeviceType  string  // 空=无
	HealthScore int16   // 0~100
	FaultCount  int32
	AgeYears    *float64 // nil=无
	Reason      string   // 空=无
	Priority    string   // MUST_REPLACE/SUGGEST/WATCH
}

// DeviceService 设备监控域服务口(阶段7):指标采集 + 健康观察。
type DeviceService interface {
	ListMetrics(ctx context.Context, resourceID int64) ([]DeviceMetric, error)
	AppendMetric(ctx context.Context, m DeviceMetric) (int64, error)
	ListMaintenances(ctx context.Context) ([]DeviceMaintenance, error)
	CreateMaintenance(ctx context.Context, m DeviceMaintenance) (int64, error)
}
