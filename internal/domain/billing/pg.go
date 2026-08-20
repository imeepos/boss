package billing

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("billing: not found")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 BillingService 接口的 PostgreSQL 实现(阶段5)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

const billCols = `id, bill_no, customer_id, customer_name, legal_entity_id, legal_entity_name, region_id, region_name, period, amount, status`

// ListBills 列出账单;customerID=0 返回全部,否则按客户过滤。
func (s *PGStore) ListBills(ctx context.Context, customerID int64) ([]Bill, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+billCols+` FROM bills WHERE ($1::bigint = 0 OR customer_id = $1) ORDER BY id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list bills: %w", err)
	}
	defer rows.Close()
	out := make([]Bill, 0)
	for rows.Next() {
		var b Bill
		if err := rows.Scan(&b.BillID, &b.BillNo, &b.CustomerID, &b.CustomerName, &b.LegalEntityID, &b.LegalEntityName,
			&b.RegionID, &b.RegionName, &b.Period, &b.Amount, &b.Status); err != nil {
			return nil, fmt.Errorf("billing: scan bill: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateBill 新建账单,返回自增 id。
func (s *PGStore) CreateBill(ctx context.Context, b Bill) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO bills(bill_no, customer_id, customer_name, legal_entity_id, legal_entity_name, region_id, region_name, period, amount, status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		b.BillNo, b.CustomerID, b.CustomerName, b.LegalEntityID, b.LegalEntityName,
		b.RegionID, b.RegionName, b.Period, b.Amount, b.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: create bill: %w", err)
	}
	return id, nil
}

// GetBill 按 id 查账单;未命中返回 ErrNotFound。
func (s *PGStore) GetBill(ctx context.Context, id int64) (*Bill, error) {
	var b Bill
	err := s.db.QueryRow(ctx, `SELECT `+billCols+` FROM bills WHERE id = $1`, id).
		Scan(&b.BillID, &b.BillNo, &b.CustomerID, &b.CustomerName, &b.LegalEntityID, &b.LegalEntityName,
			&b.RegionID, &b.RegionName, &b.Period, &b.Amount, &b.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("billing: get bill: %w", err)
	}
	return &b, nil
}

const paymentCols = `id, pay_no, COALESCE(bill_id,0), amount, method, status`

// ListPayments 列出缴费流水;billID=0 返回全部,否则按账单过滤。
func (s *PGStore) ListPayments(ctx context.Context, billID int64) ([]Payment, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+paymentCols+` FROM payments WHERE ($1 = 0 OR bill_id = $1) ORDER BY id`, billID)
	if err != nil {
		return nil, fmt.Errorf("billing: list payments: %w", err)
	}
	defer rows.Close()
	out := make([]Payment, 0)
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.PayNo, &p.BillID, &p.Amount, &p.Method, &p.Status); err != nil {
			return nil, fmt.Errorf("billing: scan payment: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreatePayment 新建缴费/充值流水,返回自增 id;billID=0(充值)落 NULL。
func (s *PGStore) CreatePayment(ctx context.Context, p Payment) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO payments(pay_no, bill_id, customer_id, amount, method, status)
		VALUES($1,NULLIF($2,0),NULLIF($3,0),$4,$5,$6) RETURNING id`,
		p.PayNo, p.BillID, p.CustomerID, p.Amount, p.Method, p.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("billing: create payment: %w", err)
	}
	return id, nil
}

// ListPaymentsByCustomer 按客户聚合流水:customer_id 直查,历史账单流水按 bills 归属兜底。
func (s *PGStore) ListPaymentsByCustomer(ctx context.Context, customerID int64) ([]Payment, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+paymentCols+` FROM payments p
		WHERE p.customer_id = $1
		   OR (p.customer_id IS NULL AND EXISTS(
		       SELECT 1 FROM bills b WHERE b.id = p.bill_id AND b.customer_id = $1))
		ORDER BY p.id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("billing: list payments by customer: %w", err)
	}
	defer rows.Close()
	out := make([]Payment, 0)
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.PayNo, &p.BillID, &p.Amount, &p.Method, &p.Status); err != nil {
			return nil, fmt.Errorf("billing: scan payment: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GenerateBills 出账:为在网客户按账期批量生成账单,幂等(ON CONFLICT DO NOTHING)。
// 金额=区域月费覆盖(region_offers 按 region_path)优先,否则产品基础月费;均为成交价快照。
func (s *PGStore) GenerateBills(ctx context.Context, period string) (int, error) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO bills(bill_no, customer_id, customer_name, legal_entity_id, legal_entity_name, region_id, region_name, period, amount, status)
		SELECT 'BILL-' || $1 || '-' || la.customer_id,
		       la.customer_id, COALESCE(c.name, ''), la.legal_entity_id, la.legal_entity_name,
		       la.region_id, la.region_name, $1, COALESCE(ro.monthly_fee, po.monthly_fee), 'UNPAID'
		FROM lo_accounts la
		JOIN customers c ON la.customer_id = c.id
		JOIN product_offers po ON la.offer_id = po.id
		LEFT JOIN region_offers ro ON ro.offer_id = la.offer_id AND la.region_path <> '' AND ro.region_path = la.region_path
		WHERE la.status = 'ACTIVE' AND c.service_status = 'ACTIVE'
		ON CONFLICT (customer_id, period) DO NOTHING`, period)
	if err != nil {
		return 0, fmt.Errorf("billing: generate bills: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
