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
