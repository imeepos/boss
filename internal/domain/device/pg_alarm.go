package device

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrForeignKeyViolation 关联实体不存在(孤儿告警防护:alarms.resource_id 软引用)。
var ErrForeignKeyViolation = errors.New("device: referenced entity not found")

// idOrNil 把 0 归一为 NULL(可空约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// exists 校验单表存在性(alarms.resource_id 无外键,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("device: check %s %d: %w", table, id, err)
	}
	return ok, nil
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
// 关联完整性:resource_id 非零时校验资源存在(曾 12 条悬空资源告警,audit
// 2026-08-25);零/NULL 保留给平台级告警(source 非 device)。
func (s *PGStore) CreateAlarm(ctx context.Context, a Alarm) (int64, error) {
	if a.ResourceID > 0 {
		ok, err := s.exists(ctx, "resources", a.ResourceID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("device: resource %d: %w", a.ResourceID, ErrForeignKeyViolation)
		}
	}
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

// AppendRetestTask 受理批量复测:落任务行,任务号 RT-YYYYMMDD-NNNN(id 后 4 位)。
func (s *PGStore) AppendRetestTask(ctx context.Context, scope string) (string, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO alarm_retest_tasks(task_no, scope) VALUES('','') RETURNING id`).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("device: append retest task: %w", err)
	}
	taskNo := fmt.Sprintf("RT-%s-%04d", time.Now().Format("20060102"), id%10000)
	if _, err := s.db.Exec(ctx,
		`UPDATE alarm_retest_tasks SET task_no=$1 WHERE id=$2`, taskNo, id); err != nil {
		return "", fmt.Errorf("device: set retest task no: %w", err)
	}
	return taskNo, nil
}
