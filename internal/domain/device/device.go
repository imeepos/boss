package device

import (
	"context"
	"time"
)

// DeviceMetric 设备监控指标(OLT 光功率/丢包率快照)。
type DeviceMetric struct {
	ID           int64     `json:"id"`
	ResourceID   int64     `json:"resourceId"`
	OpticalPower *float64  `json:"opticalPower"` // nil=离线无数据
	PacketLoss   *float64  `json:"packetLoss"`
	Status       string    `json:"status"` // ONLINE/OFFLINE/FAULT
	CollectedAt  time.Time `json:"collectedAt"`
}

// DeviceMaintenance 设备健康观察(师傅端主动运维清单)。
type DeviceMaintenance struct {
	ID          int64    `json:"id"`
	DeviceNo    string   `json:"deviceNo"`
	DeviceType  string   `json:"deviceType"`  // 空=无
	HealthScore int16    `json:"healthScore"` // 0~100
	FaultCount  int32    `json:"faultCount"`
	AgeYears    *float64 `json:"ageYears"` // nil=无
	Reason      string   `json:"reason"`   // 空=无
	Priority    string   `json:"priority"` // MUST_REPLACE/SUGGEST/WATCH
}

// DeviceService 设备监控域服务口(阶段7):指标采集 + 健康观察。
type DeviceService interface {
	ListMetrics(ctx context.Context, resourceID int64) ([]DeviceMetric, error)
	AppendMetric(ctx context.Context, m DeviceMetric) (int64, error)
	ListMaintenances(ctx context.Context) ([]DeviceMaintenance, error)
	CreateMaintenance(ctx context.Context, m DeviceMaintenance) (int64, error)
}
