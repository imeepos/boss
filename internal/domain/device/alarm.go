package device

import (
	"context"
	"time"
)

// Alarm 网络告警(网络设备/四码对账/认证异常)。
type Alarm struct {
	ID         int64     `json:"id"`
	AlarmNo    string    `json:"alarmNo"` // ALM-001
	Level      string    `json:"level"`   // CRITICAL/WARNING/INFO
	Source     string    `json:"source"`  // device/quadlink/aaa
	Content    string    `json:"content"`
	ResourceID int64     `json:"resourceId"` // 0=空(软引用 resources.id)
	Status     string    `json:"status"`     // OPEN/ACKED/CLOSED
	CreatedAt  time.Time `json:"createdAt"`
}

// AlarmService 网络监控告警域服务口(阶段7,MON)。
type AlarmService interface {
	ListAlarms(ctx context.Context, resourceID int64) ([]Alarm, error)
	CreateAlarm(ctx context.Context, a Alarm) (int64, error)
	UpdateAlarmStatus(ctx context.Context, id int64, status string) error
	// AppendRetestTask 台风应急·批量复测受理(scope=片区),返回任务号 RT-YYYYMMDD-NNNN。
	AppendRetestTask(ctx context.Context, scope string) (string, error)
}
