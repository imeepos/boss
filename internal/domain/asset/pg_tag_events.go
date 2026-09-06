// 标签绑定事件流与回收闭环(P1-T2,adopted note 2026-09-06-asset-tag-p1-wave):
// 形态依据 R2 调研(Snipe-IT action_logs / bk-cmdb cc_AuditLog)——action 短枚举、
// 差异进 changed JSONB、append-only;报废软回收禁硬删;事件仅在状态 UPDATE 命中行时写入,
// 状态层幂等天然防重放。事件写失败属于审计面损失,不阻断主状态变更,但必须
// 输出 [asset] TAG EVENT FAILED 可 grep 日志(AGENTS.md 失败路径红线)。

package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// bindTagEvent 在事务上写一条 BIND 事件(建标签预绑定/建资产绑签成功路径调用);
// 失败仅 ALERT 留痕不回滚主流程(主状态已落库,事件损失可由巡检口径补录)。
func (s *PGStore) bindTagEvent(ctx context.Context, ex ExecQuerier, tagID, assetID int64) {
	changed, _ := json.Marshal(map[string]any{"bound_asset_id": assetID})
	if _, err := ex.Exec(ctx,
		`INSERT INTO tag_events(tag_id, asset_id, action, changed) VALUES($1, $2, 'BIND', $3)`,
		tagID, assetID, changed); err != nil {
		slog.ErrorContext(ctx, "[asset] TAG EVENT FAILED",
			"action", "BIND", "tag_id", tagID, "asset_id", assetID, "err", err)
	}
}

// UnbindTag 解绑标签:置 bound_asset_id=NULL+status=UNBOUND 并写 UNBIND 事件。
// expectedAssetID>0 时校验当前绑定一致;未绑定返回 ErrTagUnbound,预期不符/并发漂移
// 返回 ErrBindingConflict。状态与事件同一事务,失败整单回滚。
func (s *PGStore) UnbindTag(ctx context.Context, tagID, expectedAssetID, actorAccountID int64, detail string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("asset: begin unbind tag: %w", err)
	}
	defer tx.Rollback(ctx)

	var bound int64
	err = tx.QueryRow(ctx,
		`SELECT COALESCE(bound_asset_id, 0) FROM tags WHERE id = $1 FOR UPDATE`, tagID).Scan(&bound)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: unbind tag lock: %w", err)
	}
	if bound == 0 {
		return ErrTagUnbound
	}
	if expectedAssetID > 0 && expectedAssetID != bound {
		return fmt.Errorf("asset: tag %d bound to %d, expect %d: %w", tagID, bound, expectedAssetID, ErrBindingConflict)
	}
	tag, err := tx.Exec(ctx,
		`UPDATE tags SET bound_asset_id = NULL, status = 'UNBOUND' WHERE id = $1 AND bound_asset_id = $2`,
		tagID, bound)
	if err != nil {
		return fmt.Errorf("asset: unbind tag: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("asset: tag %d concurrent rebind: %w", tagID, ErrBindingConflict)
	}
	changed, _ := json.Marshal(map[string]any{"bound_asset_id": []int64{bound, 0}})
	if _, err := tx.Exec(ctx,
		`INSERT INTO tag_events(tag_id, asset_id, action, actor_account_id, detail, changed) VALUES($1, $2, 'UNBIND', $3, $4, $5)`,
		tagID, bound, idOrNil(actorAccountID), detail, changed); err != nil {
		slog.ErrorContext(ctx, "[asset] TAG EVENT FAILED",
			"action", "UNBIND", "tag_id", tagID, "asset_id", bound, "err", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("asset: commit unbind tag: %w", err)
	}
	return nil
}

// ScrapAsset 报废资产:任意非终态 → SCRAPPED(终态幂等 no-op),轨迹落行;
// 标签仍绑时强制解绑并写 RECYCLE 事件(软回收禁硬删)。同一事务,失败整单回滚。
func (s *PGStore) ScrapAsset(ctx context.Context, assetID, actorAccountID int64, reason string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("asset: begin scrap: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	var tagID int64
	err = tx.QueryRow(ctx,
		`SELECT status, COALESCE(tag_id, 0) FROM assets WHERE id = $1 FOR UPDATE`, assetID).Scan(&status, &tagID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: scrap lock: %w", err)
	}
	if status == "SCRAPPED" {
		return nil // 幂等重放
	}
	if _, err := tx.Exec(ctx,
		`UPDATE assets SET status = 'SCRAPPED', updated_at = now() WHERE id = $1`, assetID); err != nil {
		return fmt.Errorf("asset: scrap: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES($1, 'SCRAPPED', now())`, assetID); err != nil {
		return fmt.Errorf("asset: scrap lifecycle: %w", err)
	}
	if tagID > 0 {
		tag, err := tx.Exec(ctx,
			`UPDATE tags SET bound_asset_id = NULL, status = 'UNBOUND' WHERE id = $1 AND bound_asset_id = $2`,
			tagID, assetID)
		if err != nil {
			return fmt.Errorf("asset: scrap recycle tag: %w", err)
		}
		if tag.RowsAffected() == 0 {
			slog.WarnContext(ctx, "[asset] TAG BIND CONFLICT",
				"tag_id", tagID, "asset_id", assetID, "reason", "scrap recycle: tag no longer bound")
			return fmt.Errorf("asset: tag %d not bound to scrapped asset %d: %w", tagID, assetID, ErrBindingConflict)
		}
		changed, _ := json.Marshal(map[string]any{"bound_asset_id": []int64{assetID, 0}})
		if _, err := tx.Exec(ctx,
			`INSERT INTO tag_events(tag_id, asset_id, action, actor_account_id, detail, changed) VALUES($1, $2, 'RECYCLE', $3, $4, $5)`,
			tagID, assetID, idOrNil(actorAccountID), reason, changed); err != nil {
			slog.ErrorContext(ctx, "[asset] TAG EVENT FAILED",
				"action", "RECYCLE", "tag_id", tagID, "asset_id", assetID, "err", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("asset: commit scrap: %w", err)
	}
	return nil
}

// ListTagEvents 标签事件流(P2-W2-T1 C):append-only 审计流,只读回放,
// 按时间倒序(created_at DESC;id DESC 兜底同秒多事件),BIND/UNBIND/RECYCLE 全量。
// 未命中标签不报错返回空列表(新标签事件流为空是合法态)。
func (s *PGStore) ListTagEvents(ctx context.Context, tagID int64) ([]TagEvent, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, event_id::text, tag_id, COALESCE(asset_id, 0), action,
		       COALESCE(actor_account_id, 0), detail, changed, created_at
		FROM tag_events WHERE tag_id = $1
		ORDER BY created_at DESC, id DESC`, tagID)
	if err != nil {
		return nil, fmt.Errorf("asset: list tag events: %w", err)
	}
	defer rows.Close()
	out := make([]TagEvent, 0)
	for rows.Next() {
		var e TagEvent
		if err := rows.Scan(&e.ID, &e.EventID, &e.TagID, &e.AssetID, &e.Action,
			&e.ActorAccountID, &e.Detail, &e.Changed, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("asset: scan tag event: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
