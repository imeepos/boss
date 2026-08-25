package provision

// W7:下发任务执行/失败/重试(状态迁移 + 全程留痕 provision_logs)。
// 装维零手工:provisioner 守护进程轮询 PENDING → ExecuteTask/ FailTask;失败可重试且计数。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrIllegalTransition 非法任务状态迁移。
var ErrIllegalTransition = errors.New("provision: illegal transition")

// ErrTaskNotFound 任务不存在。
var ErrTaskNotFound = errors.New("provision: task not found")

// ClaimTask 原子领取一个 PENDING 任务:UPDATE ... RETURNING 单语句完成
// 状态占位(DOING)与返回,FOR UPDATE SKIP LOCKED 防止多 provisioner 重复下发。
// 无待办返回 (nil, nil)。
func (s *PGStore) ClaimTask(ctx context.Context) (*Task, error) {
	var t Task
	err := s.db.QueryRow(ctx, `
		UPDATE provision_tasks SET status = 'DOING'
		WHERE id = (
			SELECT id FROM provision_tasks
			WHERE status = 'PENDING'
			ORDER BY id
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, task_no, order_id, stage_event, lo_account_id, template_id, status`,
	).Scan(&t.ID, &t.TaskNo, &t.OrderID, &t.StageEvent, &t.LoAccountID, &t.TemplateID, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("provision: claim task: %w", err)
	}
	return &t, nil
}

// ExecuteTask 完成下发任务:DOING 或 PENDING→DONE,成功写 SUCCESS 日志。
// PENDING 直接完成仅用于直调/测试;provisioner 守护进程先 ClaimTask(DOING)再执行。
func (s *PGStore) ExecuteTask(ctx context.Context, taskID int64) error {
	if err := s.transit(ctx, taskID, "DONE", "DOING", "PENDING"); err != nil {
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
