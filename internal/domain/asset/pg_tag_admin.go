// 标签域管理写侧(P2-W2-T1):建标签唯一冲突分类、禁用/启用状态机、
// DISABLED 绑定前置校验。tags.status 三态 UNBOUND/BOUND/DISABLED
// (terms.md §4 为权威;DB 侧 ck_tags_status 兜底,迁移 000184)。
package asset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
