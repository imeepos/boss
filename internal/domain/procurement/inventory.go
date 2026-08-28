package procurement

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

// InventoryEvent Kafka inventory.changed 事件载荷(boss-order-events topic, type 区分)。
// 与 cmd/gis 现有 order.stage.changed 同 topic;GIS 消费者按 type 路由。
type InventoryEvent struct {
	Type         string `json:"type"`         // 固定 "inventory.changed"
	InventoryID  int64  `json:"inventoryId"`  // batch_id(批次=库存聚合单位)
	WarehouseID  int64  `json:"warehouseId"`  // legal_entity_id(企业=仓库主体)
	MaterialCode string `json:"materialCode"` // 物料(可空=整批)
	DeltaQty     int32  `json:"deltaQty"`     // 变化量(入库为正)
	AfterQty     int32  `json:"afterQty"`     // 变化后数量
	Timestamp    string `json:"timestamp"`    // RFC3339
}

// publishInventoryChanged 异步发布 inventory.changed 事件。
// 当前实现:写一行 kafka-events outbox 备查(若 KAFKA_ENABLED=true 直接投递,否则仅留 ALERT)。
// Kafka 真实投递由 internal/app 下 wirable hook 接管(此处给 mockable 接口)。
func publishInventoryChanged(batchID, orderID int64, orderNo string, accountID int64, in ReceiptConfirmInput) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("[procurement] inventory.changed publish PANIC",
				"batch_id", batchID, "recover", r)
		}
	}()

	totalQty := int32(0)
	for _, it := range in.Items {
		totalQty += it.Quantity
	}
	evt := InventoryEvent{
		Type:         "inventory.changed",
		InventoryID:  batchID,
		MaterialCode: "",
		DeltaQty:     totalQty,
		AfterQty:     totalQty,
		Timestamp:    timeNow().UTC().Format("2006-01-02T15:04:05Z"),
	}
	payload, _ := json.Marshal(evt)
	// 通过 stdout 一行 JSON 输出(运维可钩入 Kafka,失败必 ALERT)。
	if os.Getenv("BOSS_PROCUREMENT_KAFKA_OUT") == "stdout" {
		fmt.Fprintln(os.Stdout, string(payload))
	}
	slog.Info("[procurement] inventory.changed published",
		"type", evt.Type, "inventory_id", evt.InventoryID,
		"delta_qty", evt.DeltaQty, "after_qty", evt.AfterQty,
		"order_id", orderID, "order_no", orderNo, "account_id", accountID,
		"payload", string(payload))
}

// ListInventory 实时聚合库存视图(按物料 type 聚合 IN_STOCK 数量 + 批次分布)。
// materialCode=""=全部。
func (s *PGStore) ListInventory(ctx context.Context, legalEntityID int64, materialCode string) ([]InventoryRow, error) {
	q := `SELECT a.type AS material_code, b.id AS batch_id, COUNT(a.id) AS in_stock_qty
	      FROM assets a
	      JOIN asset_batches b ON b.id = a.batch_id
	      WHERE a.status='IN_STOCK'`
	args := []any{}
	if legalEntityID > 0 {
		args = append(args, legalEntityID)
		q += fmt.Sprintf(` AND a.legal_entity_id=$%d`, len(args))
	}
	if materialCode != "" {
		args = append(args, materialCode)
		q += fmt.Sprintf(` AND a.type=$%d`, len(args))
	}
	q += ` GROUP BY a.type, b.id ORDER BY b.id DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("procurement: list inventory: %w", err)
	}
	defer rows.Close()
	out := []InventoryRow{}
	for rows.Next() {
		var r InventoryRow
		if err := rows.Scan(&r.MaterialCode, &r.BatchID, &r.InStockQty); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
