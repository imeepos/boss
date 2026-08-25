// advance 原语(环节推进单事实源)+ 环节完成广播。
package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// advance 推进一个环节:顺序守卫(stage 必须等于上一环节)+ status 迁移(经 orderSM)+ 环节日志。
// 单事实源:所有环节推进都必须过此原语,禁止直接改 stage/status。
// 推进成功后广播 order.stage.done(开放平台 Webhook,尽力而为;事件键 orderNo+stage 幂等)。
func (s *PGStore) advance(ctx context.Context, orderID int64, event string) error {
	step, ok := workflowByEvent[event]
	if !ok {
		return fmt.Errorf("order: unknown event %q", event)
	}
	var stage int8
	var status, orderNo string
	err := s.db.QueryRow(ctx, `SELECT stage, status, order_no FROM orders WHERE id = $1`, orderID).Scan(&stage, &status, &orderNo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: advance select: %w", err)
	}
	if stage != step.stage-1 {
		return ErrIllegalTransition
	}
	nextStatus := status
	if step.statusEvent != "" {
		ns, err := transition(status, step.statusEvent)
		if err != nil {
			return err
		}
		nextStatus = ns
	}
	// 前置环节完成守卫(2026-08-25 审计 §2.3.2 环节乱序防线):推进到 N(≥3)要求
	// 环节 N-1 日志行 result='DONE'。此前只看 orders.stage 计数器,环节2 PENDING
	// (资源不可用等待)时后续环节仍可推进,产生"引用有效但环节乱序"悬案(335/336/337/377/383)。
	if step.stage >= 3 {
		var prevResult string
		perr := s.db.QueryRow(ctx,
			`SELECT result FROM order_stages WHERE order_id=$1 AND stage=$2`, orderID, step.stage-1).Scan(&prevResult)
		if errors.Is(perr, pgx.ErrNoRows) {
			return ErrIllegalTransition
		}
		if perr != nil {
			return fmt.Errorf("order: advance prev-stage check: %w", perr)
		}
		if prevResult != "DONE" {
			return ErrIllegalTransition
		}
	}
	if _, err := s.db.Exec(ctx, `UPDATE orders SET stage = $2, status = $3 WHERE id = $1`, orderID, step.stage, nextStatus); err != nil {
		return fmt.Errorf("order: advance update: %w", err)
	}
	s.syncDispatchTicket(ctx, orderID, nextStatus)
	if err := s.appendStage(ctx, orderID, step.stage, "DONE"); err != nil {
		return err
	}
	s.emitStageDone(ctx, orderNo, step.stage, nextStatus)
	return nil
}

// emitStageDone 广播环节完成事件;无订阅/无注入/出错均静默(不回滚环节推进)。
func (s *PGStore) emitStageDone(ctx context.Context, orderNo string, stage int8, status string) {
	if s.notifier == nil {
		return
	}
	eventID := fmt.Sprintf("%s:stage:%d", orderNo, stage)
	_, _ = s.notifier.Emit(ctx, StageEventType, eventID, StageEventPayload{
		OrderNo: orderNo, Stage: stage, Status: status,
	})
}

// appendStage 落环节日志(order_stages)。
func (s *PGStore) appendStage(ctx context.Context, orderID int64, stage int8, result string) error {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO order_stages(order_id, stage, result) VALUES($1,$2,$3)`, orderID, stage, result); err != nil {
		return fmt.Errorf("order: append stage: %w", err)
	}
	return nil
}
