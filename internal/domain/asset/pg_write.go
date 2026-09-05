// 资产仓储写侧:批次/标签/资产/生命周期/更换/盘点/指派的创建与处理。
// 与 pg.go 读侧分文件(survey-0001 S4 整改:单文件 ≤300 行)。
package asset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgconn"
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

// CreateTag 新建电子标签,返回自增 id。
// 事务内完成:INSERT tags + 预绑定时反向回填 assets.tag_id,任一步失败整单回滚,
// 不再产生"标签已建但回填失败"的孤儿(2026-09-06 质量闸门, adopted note
// 2026-09-06-asset-tag-quality-gate)。双绑一致性口径:
// - 资产不存在 → ErrForeignKeyViolation(置备侧孤儿防御)
// - 资产已被其他标签绑定 → ErrBindingConflict(回滚整单)
// - 资产已被本标签占用 → 幂等(PG 16 命中同值仍返 1 行)
func (s *PGStore) CreateTag(ctx context.Context, t Tag) (int64, error) {
	// 预绑定前先校验资产存在(防孤儿标签)
	if t.BoundAssetID > 0 {
		ok, err := s.exists(ctx, "assets", t.BoundAssetID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("asset: bound asset %d: %w", t.BoundAssetID, ErrForeignKeyViolation)
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("asset: begin create tag: %w", err)
	}
	defer tx.Rollback(ctx) // 提交后 Rollback 为 no-op;失败路径负责回收半成品

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO tags(legal_entity_id, tag_no, epc_code, band, bound_asset_id, status, battery)
		 VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		t.LegalEntityID, t.TagNo, t.EpcCode, t.Band, idOrNil(t.BoundAssetID), t.Status, t.Battery).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create tag: %w", classifyTagInsertErr(ctx, err, t))
	}

	// 预绑定时回填 assets.tag_id;冲突即返 ErrBindingConflict 并整体回滚。
	if t.BoundAssetID > 0 {
		tag, err := tx.Exec(ctx,
			`UPDATE assets SET tag_id = $2
			 WHERE id = $1 AND (tag_id IS NULL OR tag_id = $2)`,
			t.BoundAssetID, id)
		if err != nil {
			return 0, fmt.Errorf("asset: backfill asset tag_id: %w", err)
		}
		if tag.RowsAffected() == 0 {
			// 资产.tag_id 已被其他标签占用 → 阻断隐性双绑。
			slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
				"asset_id", t.BoundAssetID, "new_tag_id", id, "tag_no", t.TagNo,
				"reason", "asset.tag_id already bound to another tag")
			return 0, fmt.Errorf("asset: asset %d already bound to another tag: %w",
				t.BoundAssetID, ErrBindingConflict)
		}
		// 绑定事件流(P1-T2):BIND 随主事务落库,失败 ALERT 不阻断。
		s.bindTagEvent(ctx, tx, id, t.BoundAssetID)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("asset: commit create tag: %w", err)
	}
	return id, nil
}

// CreateAsset 新建资产,返回自增 id。
// 事务内完成:INSERT assets + 入账轨迹首行 + tag 双向绑定回填,任一步失败
// 整单回滚,杜绝"资产已建但绑定失败"的中间态(2026-09-06 质量闸门):
// - batch/legal entity 不存在 → ErrForeignKeyViolation
// - 标签已被其他资产绑定 → ErrBindingConflict(回滚,不留孤儿资产)
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

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("asset: begin create asset: %w", err)
	}
	defer tx.Rollback(ctx) // 提交后 Rollback 为 no-op;失败路径负责回收半成品

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name,
		                    tag_id, address_id, region_id, region_name, type, status)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		a.AssetCode, a.BatchID, a.LegalEntityID, a.LegalEntityName,
		idOrNil(a.TagID), idOrNil(a.AddressID), idOrNil(a.RegionID), a.RegionName, a.Type, a.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: create asset: %w", classifyAssetInsertErr(ctx, err, a))
	}

	// 入账轨迹首行:资产生命周期从建档起完整(ITAM 惯例;历史 206 资产仅 6 条
	// 轨迹的缺口即"建档不留痕",见 adopted note 2026-09-06-asset-tag-quality-gate)。
	if _, err := tx.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, changed_at)
		 VALUES($1, $2, now())`,
		id, a.Status); err != nil {
		return 0, fmt.Errorf("asset: initial lifecycle: %w", err)
	}

	// tag 双向绑定回填:assets.tag_id 写入时同步 tags.bound_asset_id/status,
	// 与环节9 扫码核对(VerifyScan 要求 bound_asset_id 非空)口径对齐。
	// 显式比对目标值,冲突即返 ErrBindingConflict 并整体回滚。
	if a.TagID > 0 {
		tag, err := tx.Exec(ctx,
			`UPDATE tags SET bound_asset_id = $2, status = 'BOUND'
			 WHERE id = $1 AND (bound_asset_id IS NULL OR bound_asset_id = $2)`,
			a.TagID, id)
		if err != nil {
			return 0, fmt.Errorf("asset: backfill tag bound_asset_id: %w", err)
		}
		if tag.RowsAffected() == 0 {
			slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
				"tag_id", a.TagID, "new_asset_id", id, "asset_code", a.AssetCode,
				"reason", "tag.bound_asset_id already bound to another asset")
			return 0, fmt.Errorf("asset: tag %d already bound to another asset: %w",
				a.TagID, ErrBindingConflict)
		}
		// 绑定事件流(P1-T2):BIND 随主事务落库,失败 ALERT 不阻断。
		s.bindTagEvent(ctx, tx, a.TagID, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("asset: commit create asset: %w", err)
	}
	return id, nil
}

// AppendLifecycle 记录一次状态/位置变更,返回自增 id。
func (s *PGStore) AppendLifecycle(ctx context.Context, l AssetLifecycle) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, address_id, address_name, worker_id, worker_name, changed_at)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		l.AssetID, l.Status, idOrNil(l.AddressID), l.AddressName, idOrNil(l.WorkerID), l.WorkerName, l.ChangedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: append lifecycle: %w", err)
	}
	return id, nil
}

// SetAssetStatus 直改资产当前状态(换新完成联动:旧件→MAINTENANCE/新件→DEPLOYED);
// 历史轨迹由调用方 AppendLifecycle 另行落行。
func (s *PGStore) SetAssetStatus(ctx context.Context, assetID int64, status string) error {
	tag, err := s.db.Exec(ctx, `UPDATE assets SET status = $2 WHERE id = $1`, assetID, status)
	if err != nil {
		return fmt.Errorf("asset: set asset status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AssignAsset 记录一次持有(领用/部署),返回自增 id。
func (s *PGStore) AssignAsset(ctx context.Context, a AssetAssignment) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO asset_assignments(asset_id, worker_id, worker_name, address_id, address_name,
		                              reason, operator_account_id, effective_from, effective_to)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		a.AssetID, idOrNil(a.WorkerID), a.WorkerName, idOrNil(a.AddressID), a.AddressName,
		a.Reason, idOrNil(a.OperatorAccountID), a.EffectiveFrom, a.EffectiveTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("asset: assign asset: %w", err)
	}
	return id, nil
}

// classifyTagInsertErr 把 tags INSERT 23505 拆解为双绑冲突或普通唯一冲突:
//   - uq_tags_bound_asset_notnull → 双绑冲突(ErrBindingConflict)
//   - uq_tags_tag_no_key / uq_tags_epc_code_key → tag_no/epc_code 重复(原 error 透传,
//     httpx.RespondErr 不映射 23505,返回 50000;若需精确业务码后续在 httpx 增加 23505 通用映射)
//
// 其他错误原样返回。
func classifyTagInsertErr(ctx context.Context, err error, t Tag) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "uq_tags_bound_asset_notnull":
		slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
			"asset_id", t.BoundAssetID, "new_tag_no", t.TagNo,
			"reason", "DB uq_tags_bound_asset_notnull violation")
		return fmt.Errorf("asset: bound asset %d already bound to another tag: %w",
			t.BoundAssetID, ErrBindingConflict)
	}
	return err
}

// classifyAssetInsertErr 把 assets INSERT 23505 拆解为双绑冲突或普通唯一冲突:
// - uq_assets_tag_notnull → 双绑冲突(ErrBindingConflict)
// 其他错误原样返回。
func classifyAssetInsertErr(ctx context.Context, err error, a Asset) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "uq_assets_tag_notnull":
		slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
			"tag_id", a.TagID, "new_asset_code", a.AssetCode,
			"reason", "DB uq_assets_tag_notnull violation")
		return fmt.Errorf("asset: tag %d already bound to another asset: %w",
			a.TagID, ErrBindingConflict)
	}
	return err
}
