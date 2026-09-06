// 标签域管理写侧(P2-W2-T1):建标签唯一冲突分类、禁用/启用状态机、
// DISABLED 绑定前置校验。tags.status 三态 UNBOUND/BOUND/DISABLED
// (terms.md §4 为权威;DB 侧 ck_tags_status 兜底,迁移 000184)。
package asset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// classifyTagInsertErr 把 tags INSERT 23505 拆解为双绑冲突或编号/EPC 重复:
//   - uq_tags_bound_asset_notnull → 双绑冲突(ErrBindingConflict)
//   - tags_tag_no_key / tags_epc_code_key → tag_no/epc_code 重复
//     (ErrCodeDuplicate,httpx 映射 40900;P2-W2-T1 建标签端点口径:
//     编号+EPC 必填且唯一)
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
	case "tags_tag_no_key":
		slog.WarnContext(ctx, "[asset] TAG NO DUPLICATE",
			"tag_no", t.TagNo, "reason", "DB tags_tag_no_key violation")
		return fmt.Errorf("asset: tagNo %s: %w", t.TagNo, ErrCodeDuplicate)
	case "tags_epc_code_key":
		slog.WarnContext(ctx, "[asset] TAG EPC DUPLICATE",
			"epc_code", t.EpcCode, "reason", "DB tags_epc_code_key violation")
		return fmt.Errorf("asset: epcCode %s: %w", t.EpcCode, ErrCodeDuplicate)
	}
	return err
}

// QueryRower 只读查询最小接口:PGStore 主连接与 pgx.Tx 均满足,
// 供 ensureTagBindable 在库外/事务内两种上下文复用。
type QueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// ErrTagDisabled 标签已停用(DISABLED),拒绝进入绑定链路(P2-W2-T1 B)。
var ErrTagDisabled = errors.New("asset: tag disabled")

// ensureTagBindable 绑定前置校验:标签须存在且非 DISABLED。
// 接入 CreateAsset 建档绑签与 UpdateAsset 换绑两条路径(新建标签自身不可能
// DISABLED,CreateTag 预绑定无需检查);未命中 ErrForeignKeyViolation,
// DISABLED ErrTagDisabled(40900)。
func (s *PGStore) ensureTagBindable(ctx context.Context, q QueryRower, tagID int64) error {
	var status string
	err := q.QueryRow(ctx, `SELECT status FROM tags WHERE id = $1`, tagID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("asset: tag %d: %w", tagID, ErrForeignKeyViolation)
	}
	if err != nil {
		return fmt.Errorf("asset: check tag %d: %w", tagID, err)
	}
	if status == "DISABLED" {
		slog.WarnContext(ctx, "[asset] TAG BIND REJECTED",
			"tag_id", tagID, "reason", "tag DISABLED, re-enable before binding")
		return fmt.Errorf("asset: tag %d disabled: %w", tagID, ErrTagDisabled)
	}
	return nil
}

// DisableTag 停用标签:UNBOUND → DISABLED;已绑定(BOUND)拒绝,必须先解绑
// (ErrBindingConflict,40900 提示先走 /tags/{id}/unbind);已 DISABLED 幂等成功。
// 守卫 UPDATE(WHERE status='UNBOUND')+0 行回查区分原因,防并发越态。
func (s *PGStore) DisableTag(ctx context.Context, tagID int64, reason string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE tags SET status = 'DISABLED' WHERE id = $1 AND status = 'UNBOUND'`, tagID)
	if err != nil {
		return fmt.Errorf("asset: disable tag %d: %w", tagID, err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	return s.disableTagFailReason(ctx, tagID, reason)
}

// disableTagFailReason 守卫 UPDATE 0 行时区分:不存在/已停用(幂等)/仍绑定。
func (s *PGStore) disableTagFailReason(ctx context.Context, tagID int64, reason string) error {
	var status string
	var bound int64
	err := s.db.QueryRow(ctx,
		`SELECT status, COALESCE(bound_asset_id, 0) FROM tags WHERE id = $1`, tagID).
		Scan(&status, &bound)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: disable tag %d lookup: %w", tagID, err)
	}
	if status == "DISABLED" {
		_ = reason // 幂等重放
		return nil
	}
	slog.WarnContext(ctx, "[asset] TAG DISABLE REJECTED",
		"tag_id", tagID, "status", status, "bound_asset_id", bound,
		"reason", "bound tag must be unbound before disable")
	return fmt.Errorf("asset: tag %d bound to asset %d, unbind first: %w", tagID, bound, ErrBindingConflict)
}

// EnableTag 启用标签:仅对 DISABLED 生效(DISABLED → UNBOUND);
// 非 DISABLED(UNBOUND/BOUND)幂等 no-op 成功。DISABLED 只能自 UNBOUND 进入
// (BOUND 先解绑),bound_asset_id 必为 NULL,无需回查绑定。
func (s *PGStore) EnableTag(ctx context.Context, tagID int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE tags SET status = 'UNBOUND' WHERE id = $1 AND status = 'DISABLED'`, tagID)
	if err != nil {
		return fmt.Errorf("asset: enable tag %d: %w", tagID, err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var status string
	err = s.db.QueryRow(ctx, `SELECT status FROM tags WHERE id = $1`, tagID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: enable tag %d lookup: %w", tagID, err)
	}
	return nil // UNBOUND/BOUND:启用幂等 no-op
}
