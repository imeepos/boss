package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("asset: not found")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 AssetService 接口的 PostgreSQL 实现(阶段3)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// idOrNil 把 0 归一为 NULL(可空外键约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListBatches 列出全部入库批次。
func (s *PGStore) ListBatches(ctx context.Context) ([]AssetBatch, error) {
	rows, err := s.db.Query(ctx, `SELECT id, legal_entity_id, code, name FROM asset_batches ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("asset: list batches: %w", err)
	}
	defer rows.Close()
	out := make([]AssetBatch, 0)
	for rows.Next() {
		var b AssetBatch
		if err := rows.Scan(&b.ID, &b.LegalEntityID, &b.Code, &b.Name); err != nil {
			return nil, fmt.Errorf("asset: scan batch: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateBatch 新建入库批次,返回自增 id。
func (s *PGStore) CreateBatch(ctx context.Context, b AssetBatch) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO asset_batches(legal_entity_id, code, name) VALUES($1,$2,$3) RETURNING id`,
		b.LegalEntityID, b.Code, b.Name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create batch: %w", err)
	}
	return id, nil
}

// ListTags 列出全部电子标签。
func (s *PGStore) ListTags(ctx context.Context) ([]Tag, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, legal_entity_id, tag_no, epc_code, band, COALESCE(bound_asset_id, 0), status, battery
		 FROM tags ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("asset: list tags: %w", err)
	}
	defer rows.Close()
	out := make([]Tag, 0)
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.TagID, &t.LegalEntityID, &t.TagNo, &t.EpcCode, &t.Band, &t.BoundAssetID, &t.Status, &t.Battery); err != nil {
			return nil, fmt.Errorf("asset: scan tag: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTag 新建电子标签,返回自增 id。
func (s *PGStore) CreateTag(ctx context.Context, t Tag) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO tags(legal_entity_id, tag_no, epc_code, band, bound_asset_id, status, battery)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		t.LegalEntityID, t.TagNo, t.EpcCode, t.Band, idOrNil(t.BoundAssetID), t.Status, t.Battery).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create tag: %w", err)
	}
	return id, nil
}

// ListAssets 列出全部资产台账。
func (s *PGStore) ListAssets(ctx context.Context) ([]Asset, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, asset_code, batch_id, legal_entity_id, legal_entity_name,
		        COALESCE(tag_id, 0), COALESCE(address_id, 0), COALESCE(region_id, 0),
		        COALESCE(region_name, ''), type, status
		 FROM assets ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("asset: list assets: %w", err)
	}
	defer rows.Close()
	out := make([]Asset, 0)
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.AssetID, &a.AssetCode, &a.BatchID, &a.LegalEntityID, &a.LegalEntityName,
			&a.TagID, &a.AddressID, &a.RegionID, &a.RegionName, &a.Type, &a.Status); err != nil {
			return nil, fmt.Errorf("asset: scan asset: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAsset 新建资产,返回自增 id。
func (s *PGStore) CreateAsset(ctx context.Context, a Asset) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name,
		                    tag_id, address_id, region_id, region_name, type, status)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		a.AssetCode, a.BatchID, a.LegalEntityID, a.LegalEntityName,
		idOrNil(a.TagID), idOrNil(a.AddressID), idOrNil(a.RegionID), a.RegionName, a.Type, a.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create asset: %w", err)
	}
	return id, nil
}

// GetAsset 按 id 查资产;未命中返回 ErrNotFound。
func (s *PGStore) GetAsset(ctx context.Context, id int64) (*Asset, error) {
	var a Asset
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_code, batch_id, legal_entity_id, legal_entity_name,
		        COALESCE(tag_id, 0), COALESCE(address_id, 0), COALESCE(region_id, 0),
		        COALESCE(region_name, ''), type, status
		 FROM assets WHERE id = $1`, id).
		Scan(&a.AssetID, &a.AssetCode, &a.BatchID, &a.LegalEntityID, &a.LegalEntityName,
			&a.TagID, &a.AddressID, &a.RegionID, &a.RegionName, &a.Type, &a.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset: get asset: %w", err)
	}
	return &a, nil
}
