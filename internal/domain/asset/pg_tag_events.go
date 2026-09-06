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

// bindTagEvent 写一条 BIND 事件:主事务提交【之后】经 s.db 尽力而为,失败仅 ALERT
// (审计面损失不阻断主流程)。真正的落库由 bindTagEventEx 承担。
func (s *PGStore) bindTagEvent(ctx context.Context, tagID, assetID int64) {
	changed, _ := json.Marshal(map[string]any{"bound_asset_id": assetID})
	s.bindTagEventEx(ctx, s.db, tagID, assetID, string(changed))
}

// bindTagEventEx 在指定执行器(事务或 s.db)上写 BIND 事件。
// changed 必须传 string——pgx 将 []byte 按 bytea 发送,JSONB 列拒收(22P02,
// 2026-09-06 真库冒烟实证,热修见 ISSUE.md)。
func (s *PGStore) bindTagEventEx(ctx context.Context, ex ExecQuerier, tagID, assetID int64, changed string) {
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
		tagID, bound, idOrNil(actorAccountID), detail, string(changed)); err != nil {
		slog.ErrorContext(ctx, "[asset] TAG EVENT FAILED",
			"action", "UNBIND", "tag_id", tagID, "asset_id", bound, "err", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("asset: commit unbind tag: %w", err)
	}
	return nil
}

// ScrapAsset 报废资产(P1-T2+P3-F 三要素确认):任意非终态 → SCRAPPED(终态幂等 no-op),
// 轨迹落行;标签仍绑时强制解绑并写 RECYCLE 事件(软回收禁硬删)。同一事务,失败整单回滚。
// 三要素强校验防绕过前端:confirmAssetCode 须与现值精确相等;有 SN 时 confirmSn 必填相等、
// 无 SN 须空串;已绑标签时 confirmTagNo 必填相等、未绑须空串;任一不符返回
// ErrScrapConfirmMismatch(422,只指明要素不回显现值)。终态重放先于校验短路。
func (s *PGStore) ScrapAsset(ctx context.Context, assetID, actorAccountID int64, reason string, confirm ScrapConfirm) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("asset: begin scrap: %w", err)
	}
	defer tx.Rollback(ctx)

	var status, assetCode, sn string
	var tagID int64
	var tagNo *string
	err = tx.QueryRow(ctx,
		`SELECT a.status, a.asset_code, COALESCE(a.sn, ''), COALESCE(a.tag_id, 0), t.tag_no
		 FROM assets a LEFT JOIN tags t ON t.id = a.tag_id
		 WHERE a.id = $1 FOR UPDATE OF a`, assetID).
		Scan(&status, &assetCode, &sn, &tagID, &tagNo)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: scrap lock: %w", err)
	}
	if status == "SCRAPPED" {
		return nil // 幂等重放:先于三要素校验短路,同请求重放恒成功且无二次副作用
	}
	if err := verifyScrapConfirm(confirm, assetCode, sn, tagID, tagNo); err != nil {
		return err
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
			tagID, assetID, idOrNil(actorAccountID), reason, string(changed)); err != nil {
			slog.ErrorContext(ctx, "[asset] TAG EVENT FAILED",
				"action", "RECYCLE", "tag_id", tagID, "asset_id", assetID, "err", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("asset: commit scrap: %w", err)
	}
	return nil
}

// verifyScrapConfirm 三要素逐项核对:不符时返回包装 ErrScrapConfirmMismatch 的错误,
// 信息只指明不符要素与期望形态(必填/须空串),绝不回显服务端现值(防状态探测)。
func verifyScrapConfirm(confirm ScrapConfirm, assetCode, sn string, tagID int64, tagNo *string) error {
	const kind = "asset: scrap confirm 要素不符"
	if confirm.AssetCode != assetCode {
		return fmt.Errorf("%s: confirmAssetCode 与资产现值不一致: %w", kind, ErrScrapConfirmMismatch)
	}
	switch {
	case sn != "" && confirm.SN == "":
		return fmt.Errorf("%s: confirmSn 必填(该资产已登记 SN): %w", kind, ErrScrapConfirmMismatch)
	case sn == "" && confirm.SN != "":
		return fmt.Errorf("%s: confirmSn 须为空串(该资产未登记 SN): %w", kind, ErrScrapConfirmMismatch)
	case confirm.SN != sn:
		return fmt.Errorf("%s: confirmSn 与资产现值不一致: %w", kind, ErrScrapConfirmMismatch)
	}
	if tagID > 0 {
		if confirm.TagNo == "" {
			return fmt.Errorf("%s: confirmTagNo 必填(该资产已绑标签): %w", kind, ErrScrapConfirmMismatch)
		}
		if tagNo == nil || confirm.TagNo != *tagNo {
			return fmt.Errorf("%s: confirmTagNo 与标签现值不一致: %w", kind, ErrScrapConfirmMismatch)
		}
		return nil
	}
	if confirm.TagNo != "" {
		return fmt.Errorf("%s: confirmTagNo 须为空串(该资产未绑标签): %w", kind, ErrScrapConfirmMismatch)
	}
	return nil
}

// 事件查询(P2-T4 消费面):供 /tags/{tagId}/events 与 /assets/{assetId}/events 读取,
// 口径 = id 倒序 + limit(默认 50 上限 100,服务层兜底)+ action 白名单过滤 + before_id 游标。
const (
	eventListDefaultLimit = 50
	eventListMaxLimit     = 100
)

// ValidEventAction 查询过滤 action 是否在已产出动作集内(handler 拒收白名单外值)。
// CREATE 为 000189 存量回填动作(零事件资产的建档补记),时间轴回放时按此过滤。
func ValidEventAction(action string) bool {
	return action == "BIND" || action == "UNBIND" || action == "RECYCLE" || action == "CREATE"
}

// ListTagEvents 标签事件流(P2-T4):id 倒序 + limit;beforeID>0 只取更小 id(游标翻页
// 预留);actions 非空时过滤。标签不存在返回 ErrNotFound(与写侧 404 语义一致)。
func (s *PGStore) ListTagEvents(ctx context.Context, tagID, limit, beforeID int64, actions []string) ([]TagEvent, error) {
	ok, err := s.exists(ctx, "tags", tagID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return s.listEvents(ctx, "tag_id = $1", []any{tagID}, limit, beforeID, actions)
}

// ListAssetEvents 资产事件流(P2-T4):口径同 ListTagEvents;资产不存在返回 ErrNotFound。
func (s *PGStore) ListAssetEvents(ctx context.Context, assetID, limit, beforeID int64, actions []string) ([]TagEvent, error) {
	ok, err := s.exists(ctx, "assets", assetID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	return s.listEvents(ctx, "asset_id = $1", []any{assetID}, limit, beforeID, actions)
}

// listEvents 事件流共用查询:scope 固定占 $1,action 过滤/游标/limit 占位按序追加。
func (s *PGStore) listEvents(ctx context.Context, scope string, args []any, limit, beforeID int64, actions []string) ([]TagEvent, error) {
	if limit <= 0 {
		limit = eventListDefaultLimit
	}
	if limit > eventListMaxLimit {
		limit = eventListMaxLimit
	}
	where := scope
	if len(actions) > 0 {
		args = append(args, actions)
		where += fmt.Sprintf(" AND action = ANY($%d)", len(args))
	}
	if beforeID > 0 {
		args = append(args, beforeID)
		where += fmt.Sprintf(" AND id < $%d", len(args))
	}
	args = append(args, limit)
	rows, err := s.db.Query(ctx,
		"SELECT id, event_id, tag_id, COALESCE(asset_id, 0), action,"+
			" COALESCE(actor_account_id, 0), detail, changed, created_at"+
			" FROM tag_events WHERE "+where+" ORDER BY id DESC LIMIT $"+fmt.Sprint(len(args)), args...)
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
