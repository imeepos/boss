package provision

// W7:下发任务执行/失败/重试(状态迁移 + 全程留痕 provision_logs)。
// 装维零手工:provisioner 守护进程轮询 PENDING → ExecuteTask/ FailTask;失败可重试且计数。

import (
	"context"
	"errors"
	"fmt"
)

// ErrIllegalTransition 非法任务状态迁移。
var ErrIllegalTransition = errors.New("provision: illegal transition")

// ErrTaskNotFound 任务不存在。
var ErrTaskNotFound = errors.New("provision: task not found")

// ExecuteTask 执行下发任务:PENDING→DOING→DONE,成功写 SUCCESS 日志。
// 设备协议交互由 provisioner 守护进程在两次迁移之间执行;此处固化状态机与留痕。
func (s *PGStore) ExecuteTask(ctx context.Context, taskID int64) error {
	if err := s.transit(ctx, taskID, "DOING", "PENDING"); err != nil {
		return err
	}
	if err := s.transit(ctx, taskID, "DONE", "DOING"); err != nil {
		return err
	}
	_, err := s.AppendLog(ctx, Log{TaskID: taskID, Result: "SUCCESS"})
	return err
}

// FailTask 任务失败:DOING→FAILED + 失败原因留痕(result=FAILED:reason)。
func (s *PGStore) FailTask(ctx context.Context, taskID int64, reason string) error {
	if err := s.transit(ctx, taskID, "FAILED", "DOING", "PENDING"); err != nil {
		return err
	}
	_, err := s.AppendLog(ctx, Log{TaskID: taskID, Result: "FAILED: " + reason})
	return err
}

// RetryTask 失败重试:FAILED→PENDING + 重试计数留痕(可重试留痕验收点)。
func (s *PGStore) RetryTask(ctx context.Context, taskID int64, retries int16) error {
	if err := s.transit(ctx, taskID, "PENDING", "FAILED"); err != nil {
		return err
	}
	_, err := s.AppendLog(ctx, Log{TaskID: taskID, Result: "RETRY", Retries: retries + 1})
	return err
}

// transit 状态迁移:仅 from 前置态可迁;0 行命中视为非法迁移。
func (s *PGStore) transit(ctx context.Context, taskID int64, to string, from ...string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE provision_tasks SET status = $2 WHERE id = $1 AND status = ANY($3)`, taskID, to, from)
	if err != nil {
		return fmt.Errorf("provision: transit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIllegalTransition
	}
	return nil
}
