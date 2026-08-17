package device

import (
	"context"
	"fmt"
)

// idOrNil 把 0 归一为 NULL(可空约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListAlarms 列出告警;resourceID=0 返回全部,否则按设备过滤。
func (s *PGStore) ListAlarms(ctx context.Context, resourceID int64) ([]Alarm, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, alarm_no, level, source, content, COALESCE(resource_id, 0), status, created_at
		FROM alarms WHERE ($1 = 0 OR resource_id = $1) ORDER BY created_at DESC, id`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("device: list alarms: %w", err)
	}
	defer rows.Close()
	out := make([]Alarm, 0)
	for rows.Next() {
		var a Alarm
		if err := rows.Scan(&a.ID, &a.AlarmNo, &a.Level, &a.Source, &a.Content, &a.ResourceID, &a.Status, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("device: scan alarm: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAlarm 新增告警,返回自增 id。
func (s *PGStore) CreateAlarm(ctx context.Context, a Alarm) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO alarms(alarm_no, level, source, content, resource_id, status, created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		a.AlarmNo, a.Level, a.Source, a.Content, idOrNil(a.ResourceID), a.Status, a.CreatedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("device: create alarm: %w", err)
	}
	return id, nil
}

// UpdateAlarmStatus 更新告警状态(确认/关闭)。
func (s *PGStore) UpdateAlarmStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE alarms SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("device: update alarm status: %w", err)
	}
	return nil
}
