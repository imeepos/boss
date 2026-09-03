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
// 计数器(orders.stage)与留痕(order_stages)必须同事务落库:进程崩溃或单条语句失败
// 不再产生「计数器已前进而环节日志缺失」的悬案(该分叉曾导致订单永久 42200 无法续推)。
// 提交成功后广播 order.stage.done(开放平台 Webhook,尽力而为;事件键 orderNo+stage 幂等)。
func (s *PGStore) advance(ctx context.Context, orderID int64, event string) error {
	step, ok := workflowByEvent[event]
	if !ok {
		return fmt.Errorf("order: unknown event %q", event)
	}
	tdb, ok := s.db.(transactionalDB)
	if !ok {
		return fmt.Errorf("order: advance requires transaction support")
	}
	tx, err := tdb.Begin(ctx)
	if err != nil {
		return fmt.Errorf("order: advance begin: %w", err)
	}
	stage, status, orderNo, err := s.advanceGuarded(ctx, tx, orderID, step)
	if err == nil {
		if err = tx.Commit(ctx); err != nil {
			err = fmt.Errorf("order: advance commit: %w", err)
		}
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	s.emitStageDone(ctx, orderNo, stage, status)
	return nil
}

// advanceGuarded 在给定事务上完成守卫校验与迁移写入,返回提交后用于广播的订单快照。
func (s *PGStore) advanceGuarded(ctx context.Context, db dbtx, orderID int64, step stageStep) (int8, string, string, error) {
	var stage int8
	var status, orderNo string
	err := db.QueryRow(ctx, `SELECT stage, status, order_no FROM orders WHERE id = $1`, orderID).Scan(&stage, &status, &orderNo)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", "", ErrOrderNotFound
	}
	if err != nil {
		return 0, "", "", fmt.Errorf("order: advance select: %w", err)
	}
	if stage != step.stage-1 {
		return 0, "", "", ErrIllegalTransition
	}
	nextStatus := status
	if step.statusEvent != "" {
		ns, terr := transition(status, step.statusEvent)
		if terr != nil {
			return 0, "", "", terr
		}
		nextStatus = ns
	}
	// 前置环节完成守卫(2026-08-25 审计 §2.3.2 环节乱序防线):推进到 N(≥3)要求
	// 环节 N-1 日志行 result='DONE'。此前只看 orders.stage 计数器,环节2 PENDING
	// (资源不可用等待)时后续环节仍可推进,产生"引用有效但环节乱序"悬案(335/336/337/377/383)。
	if err = requirePrevStageDone(ctx, db, orderID, step.stage); err != nil {
		return 0, "", "", err
	}
	if _, err = db.Exec(ctx, `UPDATE orders SET stage = $2, status = $3 WHERE id = $1`, orderID, step.stage, nextStatus); err != nil {
		return 0, "", "", fmt.Errorf("order: advance update: %w", err)
	}
	txSync := *s
	txSync.db = db
	txSync.syncDispatchTicket(ctx, orderID, nextStatus)
	if err = appendStage(ctx, db, orderID, step.stage, "DONE"); err != nil {
		return 0, "", "", err
	}
	return step.stage, nextStatus, orderNo, nil
}

// requirePrevStageDone 前置环节日志行 result='DONE' 校验(stage<3 无前置可跳过)。
func requirePrevStageDone(ctx context.Context, db dbtx, orderID int64, stage int8) error {
	if stage < 3 {
		return nil
	}
	var prevResult string
	perr := db.QueryRow(ctx,
		`SELECT result FROM order_stages WHERE order_id=$1 AND stage=$2`, orderID, stage-1).Scan(&prevResult)
	if errors.Is(perr, pgx.ErrNoRows) {
		return ErrIllegalTransition
	}
	if perr != nil {
		return fmt.Errorf("order: advance prev-stage check: %w", perr)
	}
	if prevResult != "DONE" {
		return ErrIllegalTransition
	}
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

// appendStage 落环节日志(order_stages);db 由调用方给定(池或事务)。
// finished_at 口径(T17):推进成功(result=DONE)即写完成时刻,与 CheckResource 自愈
// UPDATE 同源(now());PENDING/DOING(等待/失败/进行中)保持 NULL——推进成功才写。
func appendStage(ctx context.Context, db dbtx, orderID int64, stage int8, result string) error {
	if _, err := db.Exec(ctx,
		`INSERT INTO order_stages(order_id, stage, result, finished_at)
		 VALUES($1,$2,$3, CASE WHEN $3 = 'DONE' THEN now() END)`, orderID, stage, result); err != nil {
		return fmt.Errorf("order: append stage: %w", err)
	}
	return nil
}
