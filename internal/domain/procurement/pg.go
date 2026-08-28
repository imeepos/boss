package procurement

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测可注入 mock。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore 是 procurement.Service 接口的 PostgreSQL 实现。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// exists 校验单表存在性。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("procurement: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// legalEntityName 反查子公司名(企业锚点字段必填,fields.md §8.1 铁律)。
func (s *PGStore) legalEntityName(ctx context.Context, id int64) (string, error) {
	var name string
	err := s.db.QueryRow(ctx, `SELECT name FROM legal_entities WHERE id=$1`, id).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("procurement: legal entity %d: %w", id, ErrForeignKey)
		}
		return "", fmt.Errorf("procurement: lookup legal entity: %w", err)
	}
	return name, nil
}

// CreateSupplier 新建供应商。
func (s *PGStore) CreateSupplier(ctx context.Context, sup Supplier) (int64, error) {
	if sup.Status == "" {
		sup.Status = "ENABLED"
	}
	if ok, err := s.exists(ctx, "legal_entities", sup.LegalEntityID); err != nil {
		return 0, err
	} else if !ok {
		return 0, fmt.Errorf("procurement: legal entity %d: %w", sup.LegalEntityID, ErrForeignKey)
	}
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO procurement_suppliers(code, name, contact_name, contact_phone,
		   legal_entity_id, status, remark)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		sup.Code, sup.Name, sup.ContactName, sup.ContactPhone,
		sup.LegalEntityID, sup.Status, sup.Remark).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("procurement: create supplier: %w", err)
	}
	return id, nil
}

// ListSuppliers 列供应商(按企业过滤;legalEntityID=0=全部)。
func (s *PGStore) ListSuppliers(ctx context.Context, legalEntityID int64) ([]Supplier, error) {
	q := `SELECT id, code, name, COALESCE(contact_name,''), COALESCE(contact_phone,''),
	             legal_entity_id, status, COALESCE(remark,''), created_at, updated_at
	      FROM procurement_suppliers`
	args := []any{}
	if legalEntityID > 0 {
		q += ` WHERE legal_entity_id=$1`
		args = append(args, legalEntityID)
	}
	q += ` ORDER BY id DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("procurement: list suppliers: %w", err)
	}
	defer rows.Close()
	out := []Supplier{}
	for rows.Next() {
		var s Supplier
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.ContactName, &s.ContactPhone,
			&s.LegalEntityID, &s.Status, &s.Remark, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DisableSupplier 禁用供应商(ENABLED→DISABLED)。
func (s *PGStore) DisableSupplier(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE procurement_suppliers SET status='DISABLED', updated_at=now() WHERE id=$1 AND status='ENABLED'`,
		id)
	if err != nil {
		return fmt.Errorf("procurement: disable supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanOrder 扫描 Order 行(Items 单独二次查询)。
func scanOrder(row pgx.Row, o *Order) error {
	return row.Scan(&o.ID, &o.ProcurementNo, &o.LegalEntityID, &o.LegalEntityName,
		&o.SupplierID, &o.SupplierName, &o.Status, &o.TotalAmount,
		&o.ExpectedDate, &o.Remark, &o.CreatedBy,
		&o.CreatedAt, &o.UpdatedAt, &o.SubmittedAt, &o.ReceivedAt, &o.CancelledAt)
}

const orderCols = `id, procurement_no, legal_entity_id, legal_entity_name,
  supplier_id, supplier_name, status, total_amount, expected_date, remark,
  created_by, created_at, updated_at, submitted_at, received_at, cancelled_at`

// GetOrder 查单笔订单(不含 Items;如需明细由 ListOrders 接口内部拼装或调用方 GetOrderItems)。
func (s *PGStore) GetOrder(ctx context.Context, id int64) (*Order, error) {
	var o Order
	err := scanOrder(s.db.QueryRow(ctx,
		`SELECT `+orderCols+` FROM procurement_orders WHERE id=$1`, id), &o)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("procurement: get order: %w", err)
	}
	return &o, nil
}

// ListOrders 列采购单(按企业 + 状态过滤;legalEntityID=0=全部)。
func (s *PGStore) ListOrders(ctx context.Context, legalEntityID int64, status string) ([]Order, error) {
	q := `SELECT ` + orderCols + ` FROM procurement_orders WHERE 1=1`
	args := []any{}
	if legalEntityID > 0 {
		args = append(args, legalEntityID)
		q += fmt.Sprintf(` AND legal_entity_id=$%d`, len(args))
	}
	if status != "" {
		args = append(args, status)
		q += fmt.Sprintf(` AND status=$%d`, len(args))
	}
	q += ` ORDER BY id DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("procurement: list orders: %w", err)
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		var o Order
		if err := scanOrder(rows, &o); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// loadOrderItems 取订单全部明细。
func (s *PGStore) loadOrderItems(ctx context.Context, orderID int64) ([]OrderItem, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, order_id, material_code, COALESCE(spec,''), quantity, received_qty,
		        unit_amount, COALESCE(remark,''), created_at, updated_at
		 FROM procurement_order_items WHERE order_id=$1 ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrderItem{}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MaterialCode, &it.Spec,
			&it.Quantity, &it.ReceivedQty, &it.UnitAmount, &it.Remark,
			&it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CreateOrder 新建采购单(头 + items 同事务);items 空数组允许,后续可补。
// 见 pg_order.go。

// CancelOrder 取消订单(非 RECEIVED 状态可取消;终态拒绝)。
func (s *PGStore) CancelOrder(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE procurement_orders SET status='CANCELLED', cancelled_at=now(), updated_at=now()
		 WHERE id=$1 AND status IN ('DRAFT','SUBMITTED','PARTIAL')`, id)
	if err != nil {
		return fmt.Errorf("procurement: cancel order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}
