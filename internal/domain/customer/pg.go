package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("customer: foreign key violation")

// ErrDuplicate 自然键重复(客户手机号/产品同公司同名等唯一约束冲突)。
var ErrDuplicate = errors.New("customer: duplicate")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// isPgUniqueViolation 唯一约束冲突(23505)。
func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// beginner 显式事务入口;*pgxpool.Pool 与 pgxmock 均满足。
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
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

// address_id 000176 起可空(先建档后补地址),读取统一 COALESCE 归零,扫描侧无感知。
const customerCols = `id, customer_code, name, phone, id_type, id_no, real_name_status, service_status, COALESCE(address_id, 0) AS address_id, legal_entity_id, region_id, region_name, created_at`

func scanCustomer(r rowScanner) (*Customer, error) {
	var c Customer
	if err := r.Scan(&c.ID, &c.CustomerCode, &c.Name, &c.Phone, &c.IdType, &c.IdNo, &c.RealNameStatus,
		&c.ServiceStatus, &c.AddressID, &c.LegalEntityID, &c.RegionID, &c.RegionName, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// exists 校验单表存在性(customers 无外键约束,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("customer: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// Create 建档,返回自增 id;customer_code 由 migration 000085 派生规则填入。
// 校验 address_id 和 legal_entity_id 存在性,防止孤儿客户。
func (s *PGStore) Create(ctx context.Context, c Customer) (int64, error) {
	// 关联完整性校验
	if c.AddressID > 0 {
		ok, err := s.exists(ctx, "addresses", c.AddressID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: address %d: %w", c.AddressID, ErrForeignKeyViolation)
		}
	}
	if c.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", c.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: legal entity %d: %w", c.LegalEntityID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO customers(name, phone, id_type, id_no, real_name_status, service_status,
		                     address_id, legal_entity_id, region_id, region_name)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id`,
		c.Name, c.Phone, c.IdType, c.IdNo, c.RealNameStatus, c.ServiceStatus,
		c.AddressID, c.LegalEntityID, c.RegionID, c.RegionName).Scan(&id)
	if isPgUniqueViolation(err) {
		return 0, ErrDuplicate
	}
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

// GetInScope 按 id 查客户,镜像 List 的数据范围语义(实体相等+region_id 落 scope 子树);
// 未命中与越界一律 ErrCustomerNotFound,保证两类情形同码同响应不可区分。
func (s *PGStore) GetInScope(ctx context.Context, id int64, legalEntityID int64, regionScope string) (*Customer, error) {
	c, err := scanCustomer(s.db.QueryRow(ctx, `
		SELECT `+customerCols+`
		FROM customers
		WHERE id = $1
		  AND ($2 = 0 OR legal_entity_id = $2)
		  AND ($3 = '' OR region_id IN (
			SELECT r.id FROM regions r
			WHERE r.path <@ text2ltree($3)
		  ))`, id, legalEntityID, regionScope))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCustomerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customer: get in scope: %w", err)
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
		  AND ($4 = 0 OR legal_entity_id = $4)
		  AND ($5 = '' OR region_id IN (
			SELECT r.id FROM regions r
			WHERE r.path <@ text2ltree($5)
		  ))
		ORDER BY id
		LIMIT $6 OFFSET $7`,
		q.NameKeyword, q.Phone, q.Status, q.LegalEntityID, q.RegionScope, limit, q.Offset)
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
