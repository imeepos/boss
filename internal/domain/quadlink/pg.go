package quadlink

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("quadlink: not found")

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("quadlink: foreign key violation")

// ErrPortAddressMismatch 端口与地址不一致(四码交叉校验)。
var ErrPortAddressMismatch = errors.New("quadlink: port-address mismatch")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 QuadLinkService 接口的 PostgreSQL 实现(阶段6)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

const linkCols = `id, COALESCE(asset_id, 0), customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status`

// ListLinks 列出全部四码关联。
func (s *PGStore) ListLinks(ctx context.Context) ([]QuadLink, error) {
	rows, err := s.db.Query(ctx, `SELECT `+linkCols+` FROM quad_links ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("quadlink: list links: %w", err)
	}
	defer rows.Close()
	out := make([]QuadLink, 0)
	for rows.Next() {
		var q QuadLink
		if err := rows.Scan(&q.ID, &q.AssetID, &q.CustomerID, &q.PortID, &q.AddressID, &q.LegalEntityID, &q.LegalEntityName, &q.Status); err != nil {
			return nil, fmt.Errorf("quadlink: scan link: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// CreateLink 新建四码关联,返回自增 id。
// 关联完整性:INSERT 前校验 customer_id/port_id/address_id 三码对应的实体均存在;
// asset_id 可为 0(预绑定阶段无资产,扫码环节9回填),为 0 时跳过资产校验,INSERT NULL;
// 任一非零 ID 不存在返回 ErrForeignKeyViolation,拒绝落"孤儿"quad_link 行。
func (s *PGStore) CreateLink(ctx context.Context, q QuadLink) (int64, error) {
	// 三码(customer/port/address)必填且实体必须存在;为 0 立即拒绝(不触 DB)。
	refs := []struct {
		table string
		id    int64
		label string
	}{
		{"customers", q.CustomerID, "customer"},
		{"ports", q.PortID, "port"},
		{"addresses", q.AddressID, "address"},
	}
	for _, r := range refs {
		if r.id == 0 {
			return 0, fmt.Errorf("quadlink: %s id required: %w", r.label, ErrForeignKeyViolation)
		}
		var exists bool
		if err := s.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM `+r.table+` WHERE id = $1)`, r.id,
		).Scan(&exists); err != nil {
			return 0, fmt.Errorf("quadlink: check %s %d: %w", r.label, r.id, err)
		}
		if !exists {
			return 0, fmt.Errorf("quadlink: %s %d not found: %w", r.label, r.id, ErrForeignKeyViolation)
		}
	}

	// asset_id:非零时校验存在性;为 0 时 INSERT NULL(扫码环节9回填)。
	if q.AssetID != 0 {
		var exists bool
		if err := s.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)`, q.AssetID,
		).Scan(&exists); err != nil {
			return 0, fmt.Errorf("quadlink: check asset %d: %w", q.AssetID, err)
		}
		if !exists {
			return 0, fmt.Errorf("quadlink: asset %d not found: %w", q.AssetID, ErrForeignKeyViolation)
		}
	}

	// legal_entity_id:运营主体必须存在(admin 侧必填,入口只查非零,存在性在此兜底)。
	var leExists bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM legal_entities WHERE id = $1)`, q.LegalEntityID,
	).Scan(&leExists); err != nil {
		return 0, fmt.Errorf("quadlink: check legal entity %d: %w", q.LegalEntityID, err)
	}
	if !leExists {
		return 0, fmt.Errorf("quadlink: legal entity %d not found: %w", q.LegalEntityID, ErrForeignKeyViolation)
	}

	// 码间交叉一致性:端口必须归属同一安装地址(ports.address_id 冗余列直查),否则四码自相矛盾。
	var portAddrMatch bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ports WHERE id = $1 AND address_id = $2)`, q.PortID, q.AddressID,
	).Scan(&portAddrMatch); err != nil {
		return 0, fmt.Errorf("quadlink: check port %d address: %w", q.PortID, err)
	}
	if !portAddrMatch {
		return 0, fmt.Errorf("quadlink: port %d not at address %d: %w", q.PortID, q.AddressID, ErrPortAddressMismatch)
	}

	var id int64
	var assetArg any // nil → INSERT NULL;int64 → INSERT value.
	if q.AssetID != 0 {
		assetArg = q.AssetID
	}
	err := s.db.QueryRow(ctx, `
		INSERT INTO quad_links(asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		assetArg, q.CustomerID, q.PortID, q.AddressID, q.LegalEntityID, q.LegalEntityName, q.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("quadlink: create link: %w", err)
	}
	return id, nil
}

// getBy 按四码之一反查(列名由调用方硬编码,非用户输入)。
// 000056 后同一码可有多行历史(UNLINKED 保留),取最新一行的当前生命周期。
func (s *PGStore) getBy(ctx context.Context, col string, val int64) (*QuadLink, error) {
	var q QuadLink
	err := s.db.QueryRow(ctx, `SELECT `+linkCols+` FROM quad_links WHERE `+col+` = $1 AND `+col+` IS NOT NULL ORDER BY id DESC LIMIT 1`, val).
		Scan(&q.ID, &q.AssetID, &q.CustomerID, &q.PortID, &q.AddressID, &q.LegalEntityID, &q.LegalEntityName, &q.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("quadlink: get by %s: %w", col, err)
	}
	return &q, nil
}

// GetByAsset 按资产反查。
func (s *PGStore) GetByAsset(ctx context.Context, assetID int64) (*QuadLink, error) {
	return s.getBy(ctx, "asset_id", assetID)
}

// GetByCustomer 按客户反查(四码第2项=客户)。
func (s *PGStore) GetByCustomer(ctx context.Context, customerID int64) (*QuadLink, error) {
	return s.getBy(ctx, "customer_id", customerID)
}

// GetByPort 按端口反查。
func (s *PGStore) GetByPort(ctx context.Context, portID int64) (*QuadLink, error) {
	return s.getBy(ctx, "port_id", portID)
}

// GetByAddress 按地址反查。
func (s *PGStore) GetByAddress(ctx context.Context, addressID int64) (*QuadLink, error) {
	return s.getBy(ctx, "address_id", addressID)
}
