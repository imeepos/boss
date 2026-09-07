package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("resource: not found")

// ErrPortNotAvailable 端口非 IDLE,无法预占。
var ErrPortNotAvailable = errors.New("resource: port not available")

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("resource: foreign key violation")

// ErrDuplicate 自然键唯一冲突(resources.code / ports.port_code / ports.quad_code)。
var ErrDuplicate = errors.New("resource: duplicate")

// isUniqueViolation 判定 PG 唯一约束冲突(23505),由 DB 索引兜底并发窗口。
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 ResourceService 接口的 PostgreSQL 实现(阶段4)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// exists 校验单表存在性(resources/ports 无外键约束,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("resource: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// idOrNil 把 0 归一为 NULL(可空外键约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

const resourceCols = `id, legal_entity_id, code, name, type, COALESCE(parent_id, 0), address_id, status`

// ListResources 列出全部网络设备。
func (s *PGStore) ListResources(ctx context.Context) ([]Resource, error) {
	rows, err := s.db.Query(ctx, `SELECT `+resourceCols+` FROM resources ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("resource: list resources: %w", err)
	}
	defer rows.Close()
	out := make([]Resource, 0)
	for rows.Next() {
		var r Resource
		if err := rows.Scan(&r.ID, &r.LegalEntityID, &r.Code, &r.Name, &r.Type, &r.ParentID, &r.AddressID, &r.Status); err != nil {
			return nil, fmt.Errorf("resource: scan resource: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateResource 新建设备,返回自增 id。
// 校验 address_id 存在性,防止孤儿设备。
func (s *PGStore) CreateResource(ctx context.Context, r Resource) (int64, error) {
	// 关联完整性校验
	if r.AddressID > 0 {
		ok, err := s.exists(ctx, "addresses", r.AddressID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("resource: address %d: %w", r.AddressID, ErrForeignKeyViolation)
		}
	}
	if r.ParentID > 0 {
		ok, err := s.exists(ctx, "resources", r.ParentID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("resource: parent %d: %w", r.ParentID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO resources(legal_entity_id, code, name, type, parent_id, address_id, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		r.LegalEntityID, r.Code, r.Name, r.Type, idOrNil(r.ParentID), r.AddressID, r.Status).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, fmt.Errorf("resource: code %s: %w", r.Code, ErrDuplicate)
		}
		return 0, fmt.Errorf("resource: create resource: %w", err)
	}
	return id, nil
}

// GetResource 按 id 查设备;未命中返回 ErrNotFound。
func (s *PGStore) GetResource(ctx context.Context, id int64) (*Resource, error) {
	var r Resource
	err := s.db.QueryRow(ctx, `SELECT `+resourceCols+` FROM resources WHERE id = $1`, id).
		Scan(&r.ID, &r.LegalEntityID, &r.Code, &r.Name, &r.Type, &r.ParentID, &r.AddressID, &r.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resource: get resource: %w", err)
	}
	return &r, nil
}

const portCols = `id, port_code, quad_code, resource_id, legal_entity_id, legal_entity_name, address_id, region_id, region_name, COALESCE(order_id, 0), status`

// GetPort 按 id 查单个端口;未命中返回 ErrNotFound。
func (s *PGStore) GetPort(ctx context.Context, portID int64) (*Port, error) {
	var p Port
	err := s.db.QueryRow(ctx, `SELECT `+portCols+` FROM ports WHERE id = $1`, portID).
		Scan(&p.PortID, &p.PortCode, &p.QuadCode, &p.ResourceID, &p.LegalEntityID, &p.LegalEntityName,
			&p.AddressID, &p.RegionID, &p.RegionName, &p.OrderID, &p.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resource: get port: %w", err)
	}
	return &p, nil
}

// ListPorts 列出端口;resourceID=0 返回全部,否则按设备过滤。
func (s *PGStore) ListPorts(ctx context.Context, resourceID int64) ([]Port, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+portCols+` FROM ports
		WHERE ($1 = 0 OR resource_id = $1) ORDER BY id`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource: list ports: %w", err)
	}
	defer rows.Close()
	out := make([]Port, 0)
	for rows.Next() {
		var p Port
		if err := rows.Scan(&p.PortID, &p.PortCode, &p.QuadCode, &p.ResourceID, &p.LegalEntityID, &p.LegalEntityName,
			&p.AddressID, &p.RegionID, &p.RegionName, &p.OrderID, &p.Status); err != nil {
			return nil, fmt.Errorf("resource: scan port: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreatePort 新建端口,返回自增 id。
// 校验 resource_id/address_id 存在性防孤儿端口;quad_code 无 DB 唯一索引,
// 预查给 40900 语义(port_code 由 DB UNIQUE 索引兜底并发窗口)。
func (s *PGStore) CreatePort(ctx context.Context, p Port) (int64, error) {
	var quadTaken bool
	if err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM ports WHERE quad_code = $1)", p.QuadCode).Scan(&quadTaken); err != nil {
		return 0, fmt.Errorf("resource: check quad_code: %w", err)
	}
	if quadTaken {
		return 0, fmt.Errorf("resource: quad_code %s: %w", p.QuadCode, ErrDuplicate)
	}
	// 关联完整性校验
	if p.ResourceID > 0 {
		ok, err := s.exists(ctx, "resources", p.ResourceID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("resource: resource %d: %w", p.ResourceID, ErrForeignKeyViolation)
		}
	}
	if p.AddressID > 0 {
		ok, err := s.exists(ctx, "addresses", p.AddressID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("resource: address %d: %w", p.AddressID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO ports(port_code, quad_code, resource_id, legal_entity_id, legal_entity_name,
		                  address_id, region_id, region_name, order_id, status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		p.PortCode, p.QuadCode, p.ResourceID, p.LegalEntityID, p.LegalEntityName,
		p.AddressID, p.RegionID, p.RegionName, idOrNil(p.OrderID), p.Status).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, fmt.Errorf("resource: port_code %s: %w", p.PortCode, ErrDuplicate)
		}
		return 0, fmt.Errorf("resource: create port: %w", err)
	}
	return id, nil
}

// ReservePort 端口预占:仅 IDLE 可预占为 RESERVED 并挂订单。
func (s *PGStore) ReservePort(ctx context.Context, portID, orderID int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'RESERVED', order_id = $2 WHERE id = $1 AND status = 'IDLE'`,
		portID, orderID)
	if err != nil {
		return fmt.Errorf("resource: reserve port: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPortNotAvailable
	}
	return nil
}

// ReserveFirstAvailable 在目标地址找一个空闲端口并预占给订单,返回端口ID。
// 原子性:单条 UPDATE 的 `AND status='IDLE'` 谓词保证并发下同一端口只被预占一次(数据库层面互斥,免 redsync)。
func (s *PGStore) ReserveFirstAvailable(ctx context.Context, addressID, orderID int64) (int64, error) {
	var portID int64
	err := s.db.QueryRow(ctx, `
		UPDATE ports SET status = 'RESERVED', order_id = $2
		WHERE id = (
			SELECT p.id FROM ports p JOIN resources r ON p.resource_id = r.id
			WHERE r.address_id = $1 AND p.status = 'IDLE'
			ORDER BY p.id LIMIT 1
		)
		AND status = 'IDLE'
		RETURNING id`, addressID, orderID).Scan(&portID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPortNotAvailable
	}
	if err != nil {
		return 0, fmt.Errorf("resource: reserve first available: %w", err)
	}
	return portID, nil
}

// ReleasePortByOrder 端口释放(取消/超时回滚预占):把挂在本订单上的 RESERVED 端口回收为 IDLE。
// 只回收 RESERVED 态,不动已占用(USED)端口;无匹配行返回 ErrPortNotAvailable。
func (s *PGStore) ReleasePortByOrder(ctx context.Context, orderID int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'IDLE', order_id = NULL
		 WHERE order_id = $1 AND status = 'RESERVED'`, orderID)
	if err != nil {
		return fmt.Errorf("resource: release port by order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPortNotAvailable
	}
	return nil
}
