package order

// 取消/释放/回退与纯状态迁移(从 pg_workflow.go 拆出,守 300 行红线)。
// 这些操作不动环节序号或重写环节日志,与 advance 正向推进正交。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Cancel 取消订单:任一未完成状态可取消(status→CANCELLED),不动环节序号;并回收预占端口。
func (s *PGStore) Cancel(ctx context.Context, orderID int64) error {
	if err := s.transitionStatus(ctx, orderID, "cancel"); err != nil {
		return err
	}
	// 取消必回收本订单预占端口(RESERVED→IDLE):否则端口死占,地址端口逐步耗尽。
	// 无预占端口(未到环节3/5)视为正常不报错;与 resource.ReleasePortByOrder 同谓词语义。
	if _, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'IDLE', order_id = NULL WHERE order_id = $1 AND status = 'RESERVED'`, orderID,
	); err != nil {
		return fmt.Errorf("order: cancel release ports: %w", err)
	}
	return nil
}

// Release 端口释放:RESERVED→PENDING(超时/取消的预占回滚),不动环节序号。
func (s *PGStore) Release(ctx context.Context, orderID int64) error {
	return s.transitionStatus(ctx, orderID, "release")
}

// RollbackStage 回退至上一完成环节(worker 端 rollback,留痕走审计):
// 删除最新环节日志 + stage 前移一位 + status 与回退后环节对齐(级联逆向:
// DONE←undone←INSTALLING←undispatch←RESERVED←release←PENDING)+ 工单随动。
// 重新推进时 advance 会重写环节日志,故删除而非标废(result 枚举无 ROLLED_BACK)。
// 返回 before/after;UPDATE 0 行(并发删除竞态)显性报错——假成功等价数据事故。
func (s *PGStore) RollbackStage(ctx context.Context, orderID int64) (int8, int8, error) {
	var stage int8
	var status string
	err := s.db.QueryRow(ctx, `SELECT stage, status FROM orders WHERE id = $1`, orderID).Scan(&stage, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return stage, stage, ErrOrderNotFound
	}
	if err != nil {
		return stage, stage, fmt.Errorf("order: rollback select: %w", err)
	}
	if stage < 2 {
		return stage, stage, ErrIllegalTransition
	}
	nextStatus, err := alignStatusToStage(status, stage-1)
	if err != nil {
		return stage, stage, err
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM order_stages WHERE order_id = $1 AND stage = $2`, orderID, stage); err != nil {
		return stage, stage, fmt.Errorf("order: rollback delete log: %w", err)
	}
	res, err := s.db.Exec(ctx,
		`UPDATE orders SET stage = $2, status = $3 WHERE id = $1`, orderID, stage-1, nextStatus)
	if err != nil {
		return stage, stage, fmt.Errorf("order: rollback update: %w", err)
	}
	if n := res.RowsAffected(); n == 0 {
		return stage, stage, fmt.Errorf("order: rollback update affected 0 rows orderID=%d", orderID)
	}
	s.syncDispatchTicket(ctx, orderID, nextStatus)
	return stage, stage - 1, nil
}

// alignStatusToStage status 逆向对齐到目标环节:status 由环节3(reserve)/8(install)/
// 11(done)产生,回退到产生环节之前则沿逆向事件级联迁移。
func alignStatusToStage(status string, target int8) (string, error) {
	next := status
	for _, r := range []struct {
		from string
		rev  string
		born int8 // 产生该 status 的环节
	}{
		{"DONE", "undone", 11},
		{"INSTALLING", "undispatch", 8},
		{"RESERVED", "release", 3},
	} {
		if next == r.from && target < r.born {
			ns, err := transition(next, r.rev)
			if err != nil {
				return "", err
			}
			next = ns
		}
	}
	return next, nil
}

// transitionStatus 只做 status 迁移(不改 stage,不写环节日志)。
func (s *PGStore) transitionStatus(ctx context.Context, orderID int64, event string) error {
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: status select: %w", err)
	}
	next, err := transition(status, event)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, orderID, next); err != nil {
		return fmt.Errorf("order: status update: %w", err)
	}
	s.syncDispatchTicket(ctx, orderID, next)
	return nil
}
