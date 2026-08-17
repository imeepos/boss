package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 CustomerService 接口的 PostgreSQL 实现(阶段2)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// rowScanner 抽象 pgx.Row 与 pgx.Rows 的 Scan,供统一扫描客户行。
type rowScanner interface {
	Scan(dest ...any) error
}

const customerCols = `id, name, phone, id_type, id_no, real_name_status, service_status, address_id, legal_entity_id, region_id, region_name, created_at`

func scanCustomer(r rowScanner) (*Customer, error) {
	var c Customer
	if err := r.Scan(&c.ID, &c.Name, &c.Phone, &c.IdType, &c.IdNo, &c.RealNameStatus,
		&c.ServiceStatus, &c.AddressID, &c.LegalEntityID, &c.RegionID, &c.RegionName, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// Create 建档,返回自增 id。
func (s *PGStore) Create(ctx context.Context, c Customer) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO customers(name, phone, id_type, id_no, real_name_status, service_status,
		                     address_id, legal_entity_id, region_id, region_name)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		c.Name, c.Phone, c.IdType, c.IdNo, c.RealNameStatus, c.ServiceStatus,
		c.AddressID, c.LegalEntityID, c.RegionID, c.RegionName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: create: %w", err)
	}
	return id, nil
}

// Get 按 id 查客户;未命中返回 ErrCustomerNotFound。
func (s *PGStore) Get(ctx context.Context, id int64) (*Customer, error) {
	c, err := scanCustomer(s.db.QueryRow(ctx,
		`SELECT `+customerCols+` FROM customers WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCustomerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customer: get: %w", err)
	}
	return c, nil
}

// List 按条件分页查询;Limit<=0 视为不限。
func (s *PGStore) List(ctx context.Context, q CustomerQuery) ([]Customer, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 1 << 30 // 等价"不限"的上限兜底
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+customerCols+`
		FROM customers
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR phone = $2)
		  AND ($3 = '' OR service_status = $3)
		ORDER BY id
		LIMIT $4 OFFSET $5`,
		q.NameKeyword, q.Phone, q.Status, limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("customer: list: %w", err)
	}
	defer rows.Close()
	out := make([]Customer, 0)
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, fmt.Errorf("customer: scan: %w", err)
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}
