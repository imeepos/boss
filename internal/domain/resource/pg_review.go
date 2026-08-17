package resource

// 调拨审批/驳回 + 手动释放预占(阶段4 台账写侧,handler 承接 oss.yaml)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrIllegalTransition 非法状态迁移(如审批非待审单)。
var ErrIllegalTransition = errors.New("resource: illegal transition")

// ApproveTransfer 调拨审批通过:PENDING→DOING;仅待审单可审批。
func (s *PGStore) ApproveTransfer(ctx context.Context, transferNo string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE transfers SET status = 'DOING' WHERE transfer_no = $1 AND status = 'PENDING'`, transferNo)
	if err != nil {
		return fmt.Errorf("resource: approve transfer: %w", err)
	}
	return reviewResult(tag)
}

// RejectTransfer 调拨驳回:PENDING→DONE(契约状态仅 PENDING/DOING/DONE,驳回即终态)。
func (s *PGStore) RejectTransfer(ctx context.Context, transferNo string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE transfers SET status = 'DONE' WHERE transfer_no = $1 AND status = 'PENDING'`, transferNo)
	if err != nil {
		return fmt.Errorf("resource: reject transfer: %w", err)
	}
	return reviewResult(tag)
}

func reviewResult(tag interface{ RowsAffected() int64 }) error {
	switch tag.RowsAffected() {
	case 0:
		return ErrNotFound
	default:
		return nil
	}
}

// ReleaseReserve 手动释放预占:行锁取记录 → 回收端口(仅 RESERVED) → 记录置 RELEASED。
// 非 HELD 记录(已释放/已核销)返回 ErrIllegalTransition,防止重复释放。
func (s *PGStore) ReleaseReserve(ctx context.Context, reserveID int64) error {
	var portID, orderID int64
	var status string
	err := s.db.QueryRow(ctx,
		`SELECT port_id, order_id, status FROM reserve_records WHERE id = $1 FOR UPDATE`, reserveID).
		Scan(&portID, &orderID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("resource: release reserve select: %w", err)
	}
	if status != "HELD" {
		return ErrIllegalTransition
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'IDLE', order_id = NULL WHERE order_id = $1 AND status = 'RESERVED'`, orderID); err != nil {
		return fmt.Errorf("resource: release reserve port: %w", err)
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE reserve_records SET status = 'RELEASED' WHERE id = $1`, reserveID); err != nil {
		return fmt.Errorf("resource: release reserve record: %w", err)
	}
	return nil
}
