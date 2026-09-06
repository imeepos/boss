package app

// 容量预警告警适配器:resource.CapacityAlarmSink → device.AlarmService(P5-W1)。
// 阈值状态即 alarms 行(source=capacity,status OPEN):判重查 OPEN,回落置 CLOSED;
// 告警域无按 source 过滤的查询,经 ListAlarms 对象级拉取后内存过滤(单对象告警量小)。

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ymm-001/boss/internal/domain/device"
)

type capacityAlarmSink struct {
	alarm device.AlarmService
}

func newCapacityAlarmSink(a device.AlarmService) capacityAlarmSink {
	return capacityAlarmSink{alarm: a}
}

// openCapacityAlarms 对象当前 OPEN 态容量告警。
func (s capacityAlarmSink) openCapacityAlarms(ctx context.Context, resourceID int64) ([]device.Alarm, error) {
	list, err := s.alarm.ListAlarms(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("capacity-sink: list alarms r%d: %w", resourceID, err)
	}
	out := make([]device.Alarm, 0)
	for _, a := range list {
		if a.Source == device.AlarmSourceCapacity && a.Status == "OPEN" {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s capacityAlarmSink) HasOpenCapacityAlarm(ctx context.Context, resourceID int64) (bool, error) {
	list, err := s.openCapacityAlarms(ctx, resourceID)
	if err != nil {
		return false, err
	}
	return len(list) > 0, nil
}

func (s capacityAlarmSink) CreateCapacityAlarm(ctx context.Context, resourceID int64, content string) error {
	_, err := s.alarm.CreateAlarm(ctx, device.Alarm{
		AlarmNo:    fmt.Sprintf("ALM-%d", time.Now().UnixNano()),
		Level:      "WARNING",
		Source:     device.AlarmSourceCapacity,
		Content:    content,
		ResourceID: resourceID,
		Status:     "OPEN",
		CreatedAt:  time.Now(),
	})
	if err != nil {
		// 失败留痕:容量告警落库失败禁止静默(可 grep 信号)。
		log.Printf("[resource-capacity] CREATE ALARM FAILED resourceId=%d content=%q err=%v", resourceID, content, err)
		return fmt.Errorf("capacity-sink: create alarm r%d: %w", resourceID, err)
	}
	return nil
}

func (s capacityAlarmSink) CloseCapacityAlarms(ctx context.Context, resourceID int64) (int64, error) {
	list, err := s.openCapacityAlarms(ctx, resourceID)
	if err != nil {
		return 0, err
	}
	var closed int64
	for _, a := range list {
		if err := s.alarm.UpdateAlarmStatus(ctx, a.ID, "CLOSED"); err != nil {
			log.Printf("[resource-capacity] CLOSE ALARM FAILED alarmId=%d resourceId=%d err=%v", a.ID, resourceID, err)
			return closed, fmt.Errorf("capacity-sink: close alarm %d: %w", a.ID, err)
		}
		closed++
	}
	return closed, nil
}
