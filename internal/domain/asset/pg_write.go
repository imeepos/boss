// 资产仓储写侧:批次/资产/生命周期/盘点/指派的创建与处理(标签写侧见 pg_tag_admin.go)。
// 与 pg.go 读侧分文件(survey-0001 S4 整改:单文件 ≤300 行)。
package asset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

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

// CreateAsset 新建资产,返回自增 id。
// 事务内完成:INSERT assets + 入账轨迹首行 + tag 双向绑定回填,任一步失败
// 整单回滚,杜绝"资产已建但绑定失败"的中间态(2026-09-06 质量闸门):
// - batch/legal entity 不存在 → ErrForeignKeyViolation
// - 标签已被其他资产绑定 → ErrBindingConflict(回滚,不留孤儿资产)
func (s *PGStore) CreateAsset(ctx context.Context, a Asset) (int64, error) {
	// 身份三要素归一(P3-T2):SN/LOID 去首尾空格,MAC 格式校验(入库原样);
	// 空串一律存 NULL(不占部分唯一索引名额)。非法 MAC ErrInvalidMAC(42200)。
	sn, mac, loid, idErr := NormalizeIdentity(a.SN, a.MAC, a.LOID)
	if idErr != nil {
		return 0, idErr
	}
	a.SN, a.MAC, a.LOID = sn, mac, loid

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
	// 企业归属快照:未显式给定时自批次回填(与采购入库建档同口径;admin 建档只传批次)。
	if a.LegalEntityID == 0 && a.BatchID > 0 {
		var leName string
		err := s.db.QueryRow(ctx,
			`SELECT b.legal_entity_id, COALESCE(le.name, '')
			 FROM asset_batches b LEFT JOIN legal_entities le ON le.id = b.legal_entity_id
			 WHERE b.id = $1`, a.BatchID).Scan(&a.LegalEntityID, &leName)
		if err != nil {
			return 0, fmt.Errorf("asset: batch %d: %w", a.BatchID, ErrForeignKeyViolation)
		}
		a.LegalEntityName = leName
	}
	// 资产编码缺省服务端生成:A-{批次 8 位}-{序号 5 位} 对齐采购入库 nextAssetCode 风格;
	// 序号取纳秒尾数防批内撞号,DB 唯一约束兜底(ErrCodeDuplicate)。
	if a.AssetCode == "" {
		a.AssetCode = genAssetCode(a.BatchID, time.Now().UTC().UnixNano()%100000)
	}
	// 型号字典(P1-T3):model_id 必须存在且未停用;Type 未显式给定时取 model.category 派生。
	if a.ModelID > 0 {
		var category string
		var active bool
		err := s.db.QueryRow(ctx, `SELECT category, is_active FROM asset_models WHERE id = $1`, a.ModelID).Scan(&category, &active)
		if err != nil {
			return 0, fmt.Errorf("asset: model %d: %w", a.ModelID, ErrForeignKeyViolation)
		}
		if !active {
			return 0, fmt.Errorf("asset: model %d deactivated: %w", a.ModelID, ErrForeignKeyViolation)
		}
		if a.Type == "" {
			a.Type = category
		}
	}
	// 类型写入白名单(P4-T2):建档最终 type(含型号派生)必须命中受控字典,
	// 白名单外 ErrTypeNotAllowed(42200),禁自由文本新方言。
	if err := ValidateType(a.Type); err != nil {
		slog.WarnContext(ctx, "[asset] TYPE NOT ALLOWED",
			"type", a.Type, "asset_code", a.AssetCode, "reason", "create: type outside whitelist")
		return 0, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("asset: begin create asset: %w", err)
	}
	defer tx.Rollback(ctx) // 提交后 Rollback 为 no-op;失败路径负责回收半成品

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO assets(asset_code, batch_id, legal_entity_id, legal_entity_name,
		                    tag_id, address_id, region_id, region_name, type, status, model_id,
		                    sn, mac, loid)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id`,
		a.AssetCode, a.BatchID, a.LegalEntityID, a.LegalEntityName,
		idOrNil(a.TagID), idOrNil(a.AddressID), idOrNil(a.RegionID), a.RegionName, a.Type, a.Status,
		idOrNil(a.ModelID), strOrNil(a.SN), strOrNil(a.MAC), strOrNil(a.LOID)).Scan(&id)
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
	bindEvent := false
	if a.TagID > 0 {
		// DISABLED 标签不可被绑定(P2-W2-T1;事务内以 tx 校验,失败整单回滚)。
		if err := s.ensureTagBindable(ctx, tx, a.TagID); err != nil {
			return 0, err
		}
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
		bindEvent = true
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("asset: commit create asset: %w", err)
	}
	// 绑定事件流(P2-T2 热修):提交后尽力而为,失败 ALERT 不阻断。
	if bindEvent {
		s.bindTagEvent(ctx, a.TagID, id)
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

// classifyAssetInsertErr 把 assets INSERT 23505 拆解为双绑冲突或普通唯一冲突:
// - uq_assets_tag_notnull → 双绑冲突(ErrBindingConflict)
// 其他错误原样返回。
func classifyAssetInsertErr(ctx context.Context, err error, a Asset) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "assets_asset_code_key":
		slog.WarnContext(ctx, "[asset] CODE DUPLICATE",
			"asset_code", a.AssetCode, "reason", "DB assets_asset_code_key violation")
		return fmt.Errorf("asset: code %s: %w", a.AssetCode, ErrCodeDuplicate)
	case "uq_assets_tag_notnull":
		slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
			"tag_id", a.TagID, "new_asset_code", a.AssetCode,
			"reason", "DB uq_assets_tag_notnull violation")
		return fmt.Errorf("asset: tag %d already bound to another asset: %w",
			a.TagID, ErrBindingConflict)
	case "uq_assets_sn", "uq_assets_mac", "uq_assets_loid":
		// 身份列唯一冲突(P3-T2):409 语义,message 携带冲突字段名(sn/mac/loid)。
		field := strings.TrimPrefix(pgErr.ConstraintName, "uq_assets_")
		slog.WarnContext(ctx, "[asset] IDENTITY DUPLICATE",
			"field", field, "asset_code", a.AssetCode,
			"reason", "DB "+pgErr.ConstraintName+" violation")
		return fmt.Errorf("asset: %s duplicate: %w", field, &ErrAssetIdentityDuplicate{Field: field})
	}
	return err
}
