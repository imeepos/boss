package procurement

import (
	"context"
	"fmt"
	"time"
)

// CreateOrder 新建采购单(头 + items 同事务);items 空数组允许,后续可补。
func (s *PGStore) CreateOrder(ctx context.Context, o Order) (int64, error) {
	if o.Status == "" {
		o.Status = "DRAFT"
	}
	if o.ProcurementNo == "" {
		o.ProcurementNo = nextProcurementNo()
	}
	if ok, err := s.exists(ctx, "procurement_suppliers", o.SupplierID); err != nil {
		return 0, err
	} else if !ok {
		return 0, fmt.Errorf("procurement: supplier %d: %w", o.SupplierID, ErrForeignKey)
	}
	name, err := s.legalEntityName(ctx, o.LegalEntityID)
	if err != nil {
		return 0, err
	}
	o.LegalEntityName = name
	if o.SupplierName == "" {
		_ = s.db.QueryRow(ctx, `SELECT name FROM procurement_suppliers WHERE id=$1`, o.SupplierID).Scan(&o.SupplierName)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("procurement: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO procurement_orders(procurement_no, legal_entity_id, legal_entity_name,
		   supplier_id, supplier_name, status, total_amount, expected_date, remark, created_by)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		o.ProcurementNo, o.LegalEntityID, o.LegalEntityName,
		o.SupplierID, o.SupplierName, o.Status, o.TotalAmount,
		o.ExpectedDate, o.Remark, o.CreatedBy).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("procurement: insert order: %w", err)
	}

	for _, it := range o.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO procurement_order_items(order_id, material_code, spec, quantity, unit_amount, remark)
			 VALUES($1,$2,$3,$4,$5,$6)`,
			id, it.MaterialCode, it.Spec, it.Quantity, it.UnitAmount, it.Remark)
		if err != nil {
			return 0, fmt.Errorf("procurement: insert item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("procurement: commit: %w", err)
	}
	return id, nil
}

// SubmitOrder 状态 DRAFT→SUBMITTED。
func (s *PGStore) SubmitOrder(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE procurement_orders SET status='SUBMITTED', submitted_at=now(), updated_at=now()
		 WHERE id=$1 AND status='DRAFT'`, id)
	if err != nil {
		return fmt.Errorf("procurement: submit order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}

// nextProcurementNo 生成采购单号 PO-YYYYMMDD-NNNNN(序列归零按日;此处按时间戳后 5 位,够用)。
// 不强求严格连续(本号非 ARN,审计可追;若需要严格流水走 redis INCR,不在本期范围)。
func nextProcurementNo() string {
	now := timeNow().UTC()
	return fmt.Sprintf("PO-%s-%05d", now.Format("20060102"), now.UnixNano()%100000)
}

// timeNow 可替换测试用(默认 time.Now UTC)。
var timeNow = func() time.Time { return time.Now() }
