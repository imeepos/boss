package device

import (
	"context"
	"time"
)

// Alarm 网络告警(网络设备/四码对账/认证异常)。
type Alarm struct {
	ID         int64
	AlarmNo    string // ALM-001
	Level      string // CRITICAL/WARNING/INFO
	Source     string // device/quadlink/aaa
	Content    string
	ResourceID int64 // 0=空(软引用 resources.id)
	Status     string // OPEN/ACKED/CLOSED
	CreatedAt  time.Time
}

// AlarmService 网络监控告警域服务口(阶段7,MON)。
type AlarmService interface {
	ListAlarms(ctx context.Context, resourceID int64) ([]Alarm, error)
	CreateAlarm(ctx context.Context, a Alarm) (int64, error)
	UpdateAlarmStatus(ctx context.Context, id int64, status string) error
}
