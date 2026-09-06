// 资产台账 admin CRUD 写侧(P2-W1-T1):受限编辑与守卫删除。
// 编辑仅开放 类型/型号/标签/批次 四键——状态与部署地址走业务流转
// (装机扫码/报废/换新),不设直改通道;删除为守卫式物理删,任一引用
// 命中即 40900 全量列明阻断项,SCRAPPED 一律拒绝硬删(软回收禁硬删)。

package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ErrBatchNotEditable 批次仅 IN_STOCK 可改(非库存态企业归属已随流转固化,直改撕裂快照链)。
var ErrBatchNotEditable = errors.New("asset: batch editable only in IN_STOCK status")

// ErrCodeDuplicate 资产编码已存在(显式指定撞号;自动生成极端碰撞同此通道)。
var ErrCodeDuplicate = errors.New("asset: asset_code already exists")

// ErrAssetReferenced 资产仍被引用禁止物理删除;Blockers 全量列出阻断项,响应可直接展示。
type ErrAssetReferenced struct{ Blockers []string }

func (e *ErrAssetReferenced) Error() string {
	return "asset: referenced by: " + strings.Join(e.Blockers, "; ")
}

// AssetUpdate 受限编辑入参(四键全量替换;BatchID 必填,TagID/ModelID 0=无,Type 空=派生或保持)。
type AssetUpdate struct {
	Type    string `json:"type"`
	ModelID int64  `json:"modelId"`
	TagID   int64  `json:"tagId"`
	BatchID int64  `json:"batchId"`
}

// UpdateAsset 受限编辑(P2-W1-T1):仅 类型/型号/标签/批次 四键,同一事务完成
// 标签换绑(旧绑 UNBIND + 新绑 BIND,冲突 ErrBindingConflict 整单回滚不留双绑)
// 与批次企业归属快照同步;批次仅 IN_STOCK 态可改;四键均无变化时零写入幂等成功。
func (s *PGStore) UpdateAsset(ctx context.Context, assetID int64, in AssetUpdate, actorAccountID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("asset: begin update asset: %w", err)
	}
	defer tx.Rollback(ctx)

	var status, atype, leName string
	var batchID, modelID, tagID, leID int64
	err = tx.QueryRow(ctx,
		`SELECT status, type, batch_id, COALESCE(model_id, 0), COALESCE(tag_id, 0), legal_entity_id, legal_entity_name
		  FROM assets WHERE id = $1 FOR UPDATE`, assetID).
		Scan(&status, &atype, &batchID, &modelID, &tagID, &leID, &leName)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: update asset lock: %w", err)
	}

	// 类型定稿:显式给定优先;新型号且未给定时由型号类别派生(与建档同口径);否则保持现值。
	newType := in.Type
	if newType == "" && in.ModelID != modelID && in.ModelID > 0 {
		var category string
		var active bool
		err := tx.QueryRow(ctx,
			`SELECT category, is_active FROM asset_models WHERE id = $1`, in.ModelID).Scan(&category, &active)
		if err != nil {
			return fmt.Errorf("asset: model %d: %w", in.ModelID, ErrForeignKeyViolation)
		}
		if !active {
			return fmt.Errorf("asset: model %d deactivated: %w", in.ModelID, ErrForeignKeyViolation)
		}
		newType = category
	} else if newType == "" {
		newType = atype
	}

	// 批次换绑:仅 IN_STOCK 可改;企业归属快照随新批次回填(与建档同口径)。
	if in.BatchID != batchID {
		if status != "IN_STOCK" {
			return fmt.Errorf("asset: asset %d status %s: %w", assetID, status, ErrBatchNotEditable)
		}
		err := tx.QueryRow(ctx,
			`SELECT b.legal_entity_id, COALESCE(le.name, '')
			  FROM asset_batches b LEFT JOIN legal_entities le ON le.id = b.legal_entity_id
			  WHERE b.id = $1`, in.BatchID).Scan(&leID, &leName)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("asset: batch %d: %w", in.BatchID, ErrForeignKeyViolation)
		}
		if err != nil {
			return fmt.Errorf("asset: update asset batch %d: %w", in.BatchID, err)
		}
	}

	// 标签换绑:先解旧绑(UNBIND)再绑新签(BIND),任一步冲突即整单回滚。
	if in.TagID != tagID {
		if err := s.rebindTagTx(ctx, tx, assetID, tagID, in.TagID, actorAccountID); err != nil {
			return err
		}
	}

	// 幂等闸门:四键均无变化时零写入成功(不触发 UPDATE 与事件)。
	if in.BatchID == batchID && in.ModelID == modelID && in.TagID == tagID && newType == atype {
		return nil
	}

	if _, err := tx.Exec(ctx,
		`UPDATE assets
		   SET type = $2, model_id = $3, tag_id = $4, batch_id = $5,
		       legal_entity_id = $6, legal_entity_name = $7, updated_at = now()
		 WHERE id = $1`,
		assetID, newType, idOrNil(in.ModelID), idOrNil(in.TagID), in.BatchID, leID, leName); err != nil {
		return fmt.Errorf("asset: update asset %d: %w", assetID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("asset: commit update asset: %w", err)
	}
	return nil
}

// rebindTagTx 事务内标签换绑:旧绑解绑写 UNBIND、新绑写 BIND;
// 新签不存在 ErrForeignKeyViolation,新签已被其他资产占用 ErrBindingConflict(整单回滚)。
func (s *PGStore) rebindTagTx(ctx context.Context, tx pgx.Tx, assetID, oldTagID, newTagID, actorAccountID int64) error {
	if oldTagID > 0 {
		tag, err := tx.Exec(ctx,
			`UPDATE tags SET bound_asset_id = NULL, status = 'UNBOUND'
			  WHERE id = $1 AND bound_asset_id = $2`, oldTagID, assetID)
		if err != nil {
			return fmt.Errorf("asset: unbind old tag %d: %w", oldTagID, err)
		}
		if tag.RowsAffected() == 0 {
			slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
				"tag_id", oldTagID, "asset_id", assetID, "reason", "update rebind: tag no longer bound to asset")
			return fmt.Errorf("asset: tag %d not bound to asset %d: %w", oldTagID, assetID, ErrBindingConflict)
		}
		changed, _ := json.Marshal(map[string]any{"bound_asset_id": []int64{assetID, 0}})
		if _, err := tx.Exec(ctx,
			`INSERT INTO tag_events(tag_id, asset_id, action, actor_account_id, changed)
			  VALUES($1, $2, 'UNBIND', $3, $4)`,
			oldTagID, assetID, idOrNil(actorAccountID), changed); err != nil {
			slog.ErrorContext(ctx, "[asset] TAG EVENT FAILED",
				"action", "UNBIND", "tag_id", oldTagID, "asset_id", assetID, "err", err)
		}
	}
	if newTagID > 0 {
		var ok bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1)`, newTagID).Scan(&ok); err != nil {
			return fmt.Errorf("asset: check tag %d: %w", newTagID, err)
		}
		if !ok {
			return fmt.Errorf("asset: tag %d: %w", newTagID, ErrForeignKeyViolation)
		}
		tag, err := tx.Exec(ctx,
			`UPDATE tags SET bound_asset_id = $2, status = 'BOUND'
			  WHERE id = $1 AND (bound_asset_id IS NULL OR bound_asset_id = $2)`, newTagID, assetID)
		if err != nil {
			return fmt.Errorf("asset: bind new tag %d: %w", newTagID, err)
		}
		if tag.RowsAffected() == 0 {
			slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
				"tag_id", newTagID, "asset_id", assetID, "reason", "update rebind: tag bound to another asset")
			return fmt.Errorf("asset: tag %d already bound to another asset: %w", newTagID, ErrBindingConflict)
		}
		// BIND 事件随主事务落库,失败 ALERT 不阻断(与建档绑定同口径)。
		s.bindTagEvent(ctx, tx, newTagID, assetID)
	}
	return nil
}

// deleteRefGuards 物理删除引用守卫清单(表,列,阻断项标签);命中任一即阻断,全量收集一次性列明。
var deleteRefGuards = []struct{ table, column, label string }{
	{"tags", "bound_asset_id", "标签绑定"},
	{"asset_assignments", "asset_id", "持有台账"},
	{"replacements", "asset_id", "换新单"},
	{"stocktake_items", "asset_id", "盘点明细"},
	{"quad_links", "asset_id", "四码关联"},
}

// genAssetCode 编码缺省生成:A-{批次 8 位}-{序号 5 位},对齐采购入库 nextAssetCode
// 风格;admin 单台建档序号取纳秒尾数,DB 唯一约束兜底撞号(ErrCodeDuplicate)。
func genAssetCode(batchID, seq int64) string {
	return fmt.Sprintf("A-%08d-%05d", batchID, seq)
}

// DeleteAsset 守卫删除(P2-W1-T1):仅 IN_STOCK 且无标签绑定/持有台账/换新单/
// 盘点明细/四码关联引用时物理删除;命中任一引用返回 ErrAssetReferenced
// (message 全量列阻断项);SCRAPPED 一律拒绝硬删提示走报废端点(软回收禁硬删)。
// 返回资产编码供审计载荷。
func (s *PGStore) DeleteAsset(ctx context.Context, assetID int64) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("asset: begin delete asset: %w", err)
	}
	defer tx.Rollback(ctx)

	var status, code string
	err = tx.QueryRow(ctx,
		`SELECT status, asset_code FROM assets WHERE id = $1 FOR UPDATE`, assetID).Scan(&status, &code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("asset: delete asset lock: %w", err)
	}
	if status == "SCRAPPED" {
		slog.WarnContext(ctx, "[asset] DELETE REJECTED",
			"asset_id", assetID, "asset_code", code, "reason", "SCRAPPED soft-recycled, hard delete forbidden")
		return "", fmt.Errorf("asset %d(%s) 已报废(SCRAPPED)禁止硬删,请走报废端点 /assets/{id}/scrap: %w", assetID, code, ErrAssetScrapped)
	}

	var blockers []string
	if status != "IN_STOCK" {
		blockers = append(blockers, "状态非 IN_STOCK("+status+")")
	}
	for _, gr := range deleteRefGuards {
		var ok bool
		// 表/列名来自包级静态白名单 deleteRefGuards,非用户输入,Sprintf 拼接安全。
		gq := "SELECT EXISTS(SELECT 1 FROM " + gr.table + " WHERE " + gr.column + " = $1)"
		err := tx.QueryRow(ctx, gq, assetID).Scan(&ok)
		if err != nil {
			return "", fmt.Errorf("asset: delete guard %s: %w", gr.table, err)
		}
		if ok {
			blockers = append(blockers, gr.label)
		}
	}
	if len(blockers) > 0 {
		return "", &ErrAssetReferenced{Blockers: blockers}
	}
	// 建档即留痕(000184):初始轨迹行是资产自有子数据,随主档同事务清理;
	// 不清则 FK asset_lifecycles_asset_id_fkey 使任何硬删必败(23503,102 E2E 实证)。
	if _, err := tx.Exec(ctx, `DELETE FROM asset_lifecycles WHERE asset_id = $1`, assetID); err != nil {
		return "", fmt.Errorf("asset: delete asset %d lifecycles: %w", assetID, err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM assets WHERE id = $1`, assetID)
	if err != nil {
		return "", fmt.Errorf("asset: delete asset %d: %w", assetID, err)
	}
	if tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("asset: commit delete asset: %w", err)
	}
	return code, nil
}
