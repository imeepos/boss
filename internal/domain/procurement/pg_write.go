package procurement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// CreateReceipt 创建入库单头(DRAFT 态;items 暂存为 0,ConfirmReceipt 时按 items 真落)。
// 入参 items 可空;真实业务流是 POST /orders/{id}/receipts 一次给齐 items + confirm。
// 此方法保留用于"先建单后补 items"的运维场景。
func (s *PGStore) CreateReceipt(ctx context.Context, r Receipt, items []ReceiptItem) (int64, error) {
	if r.ReceiptNo == "" {
		r.ReceiptNo = nextReceiptNo()
	}
	if r.Status == "" {
		r.Status = "DRAFT"
	}
	if r.OrderID <= 0 {
		return 0, fmt.Errorf("procurement: receipt order_id required: %w", ErrForeignKey)
	}
	if name, err := s.legalEntityName(ctx, r.LegalEntityID); err != nil {
		return 0, err
	} else {
		r.LegalEntityName = name
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("procurement: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO procurement_receipts(receipt_no, order_id, order_no, legal_entity_id, legal_entity_name, status)
		 VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		r.ReceiptNo, r.OrderID, r.OrderNo, r.LegalEntityID, r.LegalEntityName, r.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("procurement: insert receipt: %w", err)
	}

	for _, it := range items {
		_, err = tx.Exec(ctx,
			`INSERT INTO procurement_order_items(order_id, material_code, spec, quantity, unit_amount)
			 VALUES($1,$2,$3,$4,$5)`,
			r.OrderID, it.MaterialCode, it.Spec, it.Quantity, it.UnitAmount)
		if err != nil {
			return 0, fmt.Errorf("procurement: receipt items: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("procurement: commit: %w", err)
	}
	return id, nil
}

// ConfirmReceipt 入库确认:同事务(建 asset_batches→逐台建 assets IN_STOCK→回填 receipt)。
// 触发 inventory.changed 事件(Kafka topic=`boss-order-events`, type=`inventory.changed`)。
//
// 失败路径:[procurement] CONFIRM RECEIPT FAILED 级 ALERT 日志,必带载荷上下文(receipt_id/order_id/items)。
func (s *PGStore) ConfirmReceipt(ctx context.Context, receiptID, accountID int64, in ReceiptConfirmInput) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("procurement: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var orderID int64
	var orderNo, legalEntityName, status string
	err = tx.QueryRow(ctx,
		`SELECT r.order_id, o.procurement_no, r.legal_entity_name, r.status
		 FROM procurement_receipts r JOIN procurement_orders o ON o.id = r.order_id
		 WHERE r.id=$1 FOR UPDATE`, receiptID).Scan(&orderID, &orderNo, &legalEntityName, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("procurement: lock receipt: %w", err)
	}
	if status != "DRAFT" {
		return fmt.Errorf("procurement: receipt %d status=%s: %w", receiptID, status, ErrInvalidTransition)
	}

	batchCode := in.BatchCode
	if batchCode == "" {
		batchCode = nextBatchCode(orderNo)
	}

	var batchID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO asset_batches(legal_entity_id, code, name, warehouse_lat, warehouse_lng)
		 SELECT r.legal_entity_id, $2, $3, $4, $5
		 FROM procurement_receipts r WHERE r.id=$1 RETURNING id`,
		receiptID, batchCode, in.BatchName, in.WarehouseLat, in.WarehouseLng).Scan(&batchID)
	if err != nil {
		slog.ErrorContext(ctx, "[procurement] CONFIRM RECEIPT FAILED",
			"stage", "insert_asset_batch", "receipt_id", receiptID, "err", err)
		return fmt.Errorf("procurement: insert asset_batches: %w", err)
	}

	// 逐台建 assets(IN_STOCK);物料 type=光猫/ONU 用 material_items 主档对齐(L0),简化以 materialCode 直接写 type 字段。
	for _, it := range in.Items {
		if it.Quantity <= 0 {
			continue
		}
		for i := int32(0); i < it.Quantity; i++ {
			code := fmt.Sprintf("A-%s-%s-%05d", batchCode, it.MaterialCode, i+1)
			_, err = tx.Exec(ctx,
				`INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name, type, status)
				 SELECT $1, $2, r.legal_entity_id, r.legal_entity_name, $3, 'IN_STOCK'
				 FROM procurement_receipts r WHERE r.id=$4`,
				code, batchID, it.MaterialCode, receiptID)
			if err != nil {
				slog.ErrorContext(ctx, "[procurement] CONFIRM RECEIPT FAILED",
					"stage", "insert_asset", "receipt_id", receiptID, "batch_id", batchID,
					"material", it.MaterialCode, "seq", i, "err", err)
				return fmt.Errorf("procurement: insert asset: %w", err)
			}
		}
		// 同步 order_items.received_qty(累加)
		_, err = tx.Exec(ctx,
			`UPDATE procurement_order_items SET received_qty = received_qty + $2, updated_at = now()
			 WHERE order_id = $1 AND material_code = $3`,
			orderID, it.Quantity, it.MaterialCode)
		if err != nil {
			return fmt.Errorf("procurement: bump received_qty: %w", err)
		}
	}

	// 全量收齐?子查询比,RECEIVED 状态从 PARTIAL/SUBMITTED 推进。
	var ordered, received int64
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity),0), COALESCE(SUM(received_qty),0)
		 FROM procurement_order_items WHERE order_id=$1`, orderID).Scan(&ordered, &received)
	if err != nil {
		return fmt.Errorf("procurement: sum items: %w", err)
	}
	newOrderStatus := "PARTIAL"
	if received >= ordered {
		newOrderStatus = "RECEIVED"
	}
	_, err = tx.Exec(ctx,
		`UPDATE procurement_orders SET status=$2, received_at = CASE WHEN $2='RECEIVED' THEN now() ELSE received_at END, updated_at=now()
		 WHERE id=$1 AND status IN ('SUBMITTED','PARTIAL')`, orderID, newOrderStatus)
	if err != nil {
		return fmt.Errorf("procurement: advance order status: %w", err)
	}

	// 回填 receipt
	_, err = tx.Exec(ctx,
		`UPDATE procurement_receipts SET status='CONFIRMED', batch_id=$2, received_by=$3, received_at=now(), remark=$4, updated_at=now()
		 WHERE id=$1`, receiptID, batchID, accountID, in.BatchName)
	if err != nil {
		return fmt.Errorf("procurement: update receipt: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "[procurement] CONFIRM RECEIPT FAILED",
			"stage", "commit", "receipt_id", receiptID, "err", err)
		return fmt.Errorf("procurement: commit: %w", err)
	}

	// 事务提交后异步广播 inventory.changed;失败仅 ALERT 留痕(下游消费器自愈幂等)。
	go publishInventoryChanged(batchID, orderID, orderNo, accountID, in)
	slog.InfoContext(ctx, "[procurement] CONFIRM RECEIPT OK",
		"receipt_id", receiptID, "order_id", orderID, "batch_id", batchID, "items", len(in.Items))
	return nil
}

// nextReceiptNo RC-YYYYMMDD-NNNNN(序列归零按日)。
func nextReceiptNo() string {
	now := timeNow().UTC()
	return fmt.Sprintf("RC-%s-%05d", now.Format("20060102"), now.UnixNano()%100000)
}

// nextBatchCode RK-YYYYMMDD-NNNNN(入库批次编码,沿用 asset_batches.code 习惯)。
func nextBatchCode(orderNo string) string {
	now := timeNow().UTC()
	return fmt.Sprintf("RK-%s-%s", now.Format("20060102"), orderNo)
}

// ListReceipts 列入库单(按 order 过滤;orderID=0=全部)。
func (s *PGStore) ListReceipts(ctx context.Context, orderID int64) ([]Receipt, error) {
	q := `SELECT id, receipt_no, order_id, order_no, batch_id, legal_entity_id, legal_entity_name,
	             received_by, received_at, status, COALESCE(remark,''), created_at, updated_at
	      FROM procurement_receipts WHERE 1=1`
	args := []any{}
	if orderID > 0 {
		args = append(args, orderID)
		q += fmt.Sprintf(` AND order_id=$%d`, len(args))
	}
	q += ` ORDER BY id DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("procurement: list receipts: %w", err)
	}
	defer rows.Close()
	out := []Receipt{}
	for rows.Next() {
		var r Receipt
		if err := rows.Scan(&r.ID, &r.ReceiptNo, &r.OrderID, &r.OrderNo, &r.BatchID,
			&r.LegalEntityID, &r.LegalEntityName, &r.ReceivedBy, &r.ReceivedAt,
			&r.Status, &r.Remark, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
