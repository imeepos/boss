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

const linkCols = `id, asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status`

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
func (s *PGStore) CreateLink(ctx context.Context, q QuadLink) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO quad_links(asset_id, customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		q.AssetID, q.CustomerID, q.PortID, q.AddressID, q.LegalEntityID, q.LegalEntityName, q.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("quadlink: create link: %w", err)
	}
	return id, nil
}

// getBy 按四码之一反查(列名由调用方硬编码,非用户输入)。
// 000056 后同一码可有多行历史(UNLINKED 保留),取最新一行的当前生命周期。
func (s *PGStore) getBy(ctx context.Context, col string, val int64) (*QuadLink, error) {
	var q QuadLink
	err := s.db.QueryRow(ctx, `SELECT `+linkCols+` FROM quad_links WHERE `+col+` = $1 ORDER BY id DESC LIMIT 1`, val).
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
