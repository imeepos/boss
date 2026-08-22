// 资产仓储写侧:批次/标签/资产/生命周期/更换/盘点/指派的创建与处理。
// 与 pg.go 读侧分文件(survey-0001 S4 整改:单文件 ≤300 行)。
package asset

import (
	"context"
	"fmt"
)

// CreateBatch 新建入库批次,返回自增 id。
// 校验 legal_entity_id 存在性,防止孤儿批次。
func (s *PGStore) CreateBatch(ctx context.Context, b AssetBatch) (int64, error) {
	// 关联完整性校验
	if b.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", b.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("asset: legal entity %d: %w", b.LegalEntityID, ErrForeignKeyViolation)
		}
	}

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

// CreateAsset 新建资产,返回自增 id。
// 校验 batch_id 和 legal_entity_id 存在性,防止孤儿资产。
func (s *PGStore) CreateAsset(ctx context.Context, a Asset) (int64, error) {
	// 关联完整性校验
	if a.BatchID > 0 {
		ok, err := s.exists(ctx, "asset_batches", a.BatchID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("asset: batch %d: %w", a.BatchID, ErrForeignKeyViolation)
		}
	}
	if a.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", a.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("asset: legal entity %d: %w", a.LegalEntityID, ErrForeignKeyViolation)
		}
	}

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
	// tag 双向绑定回填:assets.tag_id 写入时同步 tags.bound_asset_id/status,
	// 与环节9 扫码核对(VerifyScan 要求 bound_asset_id 非空)口径对齐。
	if a.TagID > 0 {
		if _, err := s.db.Exec(ctx,
			`UPDATE tags SET bound_asset_id = $2, status = 'BOUND'
			 WHERE id = $1 AND bound_asset_id IS NULL`, a.TagID, id); err != nil {
			return 0, fmt.Errorf("asset: bind tag: %w", err)
		}
	}
	return id, nil
}

// GetAsset 按 id 查资产;未命中返回 ErrNotFound。

// AppendLifecycle 记录一次状态/位置变更,返回自增 id。
func (s *PGStore) AppendLifecycle(ctx context.Context, l AssetLifecycle) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO asset_lifecycles(asset_id, status, address_id, address_name, worker_id, worker_name, changed_at)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		l.AssetID, l.Status, idOrNil(l.AddressID), l.AddressName, idOrNil(l.WorkerID), l.WorkerName, l.ChangedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: append lifecycle: %w", err)
	}
	return id, nil
}

// ListReplacements 列出全部换新单。

// CreateReplacement 新建换新单,返回自增 id。
func (s *PGStore) CreateReplacement(ctx context.Context, r Replacement) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO replacements(replacement_no, asset_id, legal_entity_id, legal_entity_name, reason, priority, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		r.ReplacementNo, r.AssetID, r.LegalEntityID, r.LegalEntityName, r.Reason, r.Priority, r.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create replacement: %w", err)
	}
	return id, nil
}

// ListStocktakes 列出全部盘点任务。

// CreateStocktake 新建盘点任务,返回自增 id。
func (s *PGStore) CreateStocktake(ctx context.Context, st Stocktake) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO stocktakes(legal_entity_id, scope, progress, diff_count, status)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		st.LegalEntityID, st.Scope, st.Progress, st.DiffCount, st.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create stocktake: %w", err)
	}
	return id, nil
}

// HandleStocktakeDiff 盘点差异项处理(asset.yaml handleStocktakeDiff):差异处理完任务置 DONE。

// HandleStocktakeDiff 盘点差异项处理(asset.yaml handleStocktakeDiff):差异处理完任务置 DONE。
func (s *PGStore) HandleStocktakeDiff(ctx context.Context, taskID int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE stocktakes SET status = 'DONE' WHERE id = $1 AND status = 'DOING'`, taskID)
	if err != nil {
		return fmt.Errorf("asset: handle stocktake diff: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAssignments 列出资产持有台账,按生效时间升序。

// AssignAsset 记录一次持有(领用/部署),返回自增 id。
func (s *PGStore) AssignAsset(ctx context.Context, a AssetAssignment) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO asset_assignments(asset_id, worker_id, worker_name, address_id, address_name,
		                              reason, operator_account_id, effective_from, effective_to)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		a.AssetID, idOrNil(a.WorkerID), a.WorkerName, idOrNil(a.AddressID), a.AddressName,
		a.Reason, idOrNil(a.OperatorAccountID), a.EffectiveFrom, a.EffectiveTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: assign asset: %w", err)
	}
	return id, nil
}
