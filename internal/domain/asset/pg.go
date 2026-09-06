package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("asset: not found")

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("asset: foreign key violation")

// ErrTagUnbound 标签当前未绑定任何资产(P1-T2 解绑前置校验)。
var ErrTagUnbound = errors.New("asset: tag not bound")

// ErrModelExists 型号字典重复(vendor+model+category+part_number 已存在,P1-T3)。
var ErrModelExists = errors.New("asset: model already exists")

// ErrBindingConflict 资产/标签双绑冲突:目标已被另一方绑定。
// 用于 POST /provision/{assets,tags} 同步回填时,反向记录已被占用的场景。
var ErrBindingConflict = errors.New("asset: tag-asset binding conflict")

// ErrScrapConfirmMismatch 报废三要素确认不符(P3-F):资产编码/SN/标签号任一不匹配、
// 该填不填或该空不空。422 语义:错误信息只指明哪个要素不符,不回显服务端现值(防状态探测)。
var ErrScrapConfirmMismatch = errors.New("asset: scrap confirm mismatch")

// ErrDiffPending 盘点存在未处置差异,禁止关单。
var ErrDiffPending = errors.New("asset: stocktake diff items pending")

// ErrStocktakeState 盘点任务/明细状态不允许该操作(已关单/已处置/无需处置)。
var ErrStocktakeState = errors.New("asset: stocktake state invalid")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
// Begin 供 CreateAsset/CreateTag 事务化写侧使用(失败整单回滚)。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore 是 AssetService 接口的 PostgreSQL 实现(阶段3)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// exists 校验单表存在性(assets 无外键约束,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("asset: check %s %d: %w", table, id, err)
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

// ListAssetsPage/ListTagsPage 分页列表见 pg_page.go(P3-T1,取代全量 ListAssets)。

// CreateAsset 新建资产,返回自增 id。

// GetAsset 按 id 查资产;未命中返回 ErrNotFound。
func (s *PGStore) GetAsset(ctx context.Context, id int64) (*Asset, error) {
	var a Asset
	err := s.db.QueryRow(ctx,
		`SELECT id, asset_code, batch_id, legal_entity_id, legal_entity_name,
		        COALESCE(tag_id, 0), COALESCE(address_id, 0), COALESCE(region_id, 0),
		        COALESCE(region_name, ''), type, status, COALESCE(model_id, 0),
		        COALESCE(sn, ''), COALESCE(mac, ''), COALESCE(loid, '')
		 FROM assets WHERE id = $1`, id).
		Scan(&a.AssetID, &a.AssetCode, &a.BatchID, &a.LegalEntityID, &a.LegalEntityName,
			&a.TagID, &a.AddressID, &a.RegionID, &a.RegionName, &a.Type, &a.Status, &a.ModelID,
			&a.SN, &a.MAC, &a.LOID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset: get asset: %w", err)
	}
	return &a, nil
}

// ListLifecycles 列出资产状态轨迹,按变更时间升序。

// ListLifecycles 列出资产状态轨迹,按变更时间升序。
func (s *PGStore) ListLifecycles(ctx context.Context, assetID int64) ([]AssetLifecycle, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, asset_id, status, COALESCE(address_id, 0), COALESCE(address_name, ''),
		       COALESCE(worker_id, 0), COALESCE(worker_name, ''), changed_at
		FROM asset_lifecycles WHERE asset_id = $1 ORDER BY changed_at, id`, assetID)
	if err != nil {
		return nil, fmt.Errorf("asset: list lifecycles: %w", err)
	}
	defer rows.Close()
	out := make([]AssetLifecycle, 0)
	for rows.Next() {
		var l AssetLifecycle
		if err := rows.Scan(&l.ID, &l.AssetID, &l.Status, &l.AddressID, &l.AddressName, &l.WorkerID, &l.WorkerName, &l.ChangedAt); err != nil {
			return nil, fmt.Errorf("asset: scan lifecycle: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AppendLifecycle 记录一次状态/位置变更,返回自增 id。

// ListReplacements/CreateReplacement/状态机流转见 pg_replacement.go(单文件 ≤300 行)。

// ListStocktakes 列出全部盘点任务。
func (s *PGStore) ListStocktakes(ctx context.Context) ([]Stocktake, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, legal_entity_id, scope, progress, diff_count, status FROM stocktakes ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("asset: list stocktakes: %w", err)
	}
	defer rows.Close()
	out := make([]Stocktake, 0)
	for rows.Next() {
		var st Stocktake
		if err := rows.Scan(&st.ID, &st.LegalEntityID, &st.Scope, &st.Progress, &st.DiffCount, &st.Status); err != nil {
			return nil, fmt.Errorf("asset: scan stocktake: %w", err)
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// CreateStocktake 新建盘点任务,返回自增 id。

// ListAssignments 列出资产持有台账,按生效时间升序。
func (s *PGStore) ListAssignments(ctx context.Context, assetID int64) ([]AssetAssignment, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, asset_id, COALESCE(worker_id, 0), COALESCE(worker_name, ''),
		       COALESCE(address_id, 0), COALESCE(address_name, ''),
		       COALESCE(reason, ''), COALESCE(operator_account_id, 0),
		       effective_from, effective_to
		FROM asset_assignments WHERE asset_id = $1 ORDER BY effective_from, id`, assetID)
	if err != nil {
		return nil, fmt.Errorf("asset: list assignments: %w", err)
	}
	defer rows.Close()
	out := make([]AssetAssignment, 0)
	for rows.Next() {
		var a AssetAssignment
		var effTo pgtype.Timestamptz
		if err := rows.Scan(&a.ID, &a.AssetID, &a.WorkerID, &a.WorkerName, &a.AddressID, &a.AddressName,
			&a.Reason, &a.OperatorAccountID, &a.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("asset: scan assignment: %w", err)
		}
		if effTo.Valid {
			t := effTo.Time
			a.EffectiveTo = &t
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AssignAsset 记录一次持有(领用/部署),返回自增 id。
