package procurement

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetOrderDetail 采购单详情(D):单头+明细行+状态;未命中沿用既有 ErrNotFound(40400)。
func (s *PGStore) GetOrderDetail(ctx context.Context, id int64) (*Order, error) {
	o, err := s.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := s.loadOrderItems(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("procurement: order %d items: %w", id, err)
	}
	o.Items = items
	return o, nil
}

// RejectReceipt 入库驳回(E):仅 DRAFT 可以驳回,置 REJECTED;CONFIRMED 等非 DRAFT
// 一律 ErrStateConflict(HTTP 40900);原因限 255 字(超限 ErrInvalidInput),
// 写回 receipt.remark 供列表展示;操作人/原因审计由 admin handler 层 RecordAudit 补记。
func (s *PGStore) RejectReceipt(ctx context.Context, receiptID int64, reason string) error {
	if len(reason) > 255 {
		return fmt.Errorf("procurement: receipt %d reject reason %d bytes: %w", receiptID, len(reason), ErrInvalidInput)
	}
	var status string
	err := s.db.QueryRow(ctx,
		`SELECT status FROM procurement_receipts WHERE id=$1`, receiptID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("procurement: lookup receipt: %w", err)
	}
	if status != "DRAFT" {
		return fmt.Errorf("procurement: receipt %d status=%s: %w", receiptID, status, ErrStateConflict)
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE procurement_receipts SET status='REJECTED', remark=$2, updated_at=now()
		 WHERE id=$1 AND status='DRAFT'`, receiptID, reason)
	if err != nil {
		return fmt.Errorf("procurement: reject receipt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("procurement: receipt %d status=%s: %w", receiptID, status, ErrStateConflict)
	}
	return nil
}
