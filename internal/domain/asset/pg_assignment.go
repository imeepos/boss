// 资产持有台账写侧(P2-W2-T1 G/H):领用开段/归还闭段。
// 台账模型:每次领用一行,effective_from→effective_to 时间段,历史不随当前值漂移
// (fields.md §8 归属台账通用模式)。
package asset

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrAssetNotInStock 资产非库存态,禁止领用(P2-W2-T1 G,40900)。
var ErrAssetNotInStock = errors.New("asset: not in stock")

// CreateAssignment 领用(P2-W2-T1 G):资产/师傅/事由必填(handler 层校验);
// 仅 IN_STOCK 可领用(否则 ErrAssetNotInStock,40900);落持有台账开段
// (effective_from=now,effective_to NULL)。师傅存在性校验防孤儿段。
//
// 资产状态口径(硬性裁定):领用【不改】资产状态——既有装机流程(环节9 扫码
// MATCH,000185 资产联动)才置 DEPLOYED;领用是"师傅从库存领走"的台账事实,
// 资产保持 IN_STOCK,DEPLOYED 随装机流转(与 fields.md §4.1 口径一致)。
func (s *PGStore) CreateAssignment(ctx context.Context, a AssetAssignment) (int64, error) {
	ast, err := s.GetAsset(ctx, a.AssetID)
	if err != nil {
		return 0, fmt.Errorf("asset: assignment asset %d: %w", a.AssetID, ErrForeignKeyViolation)
	}
	if ast.Status != "IN_STOCK" {
		return 0, fmt.Errorf("asset: asset %d status %s: %w", a.AssetID, ast.Status, ErrAssetNotInStock)
	}
	if ok, err := s.exists(ctx, "workers", a.WorkerID); err != nil {
		return 0, err
	} else if !ok {
		return 0, fmt.Errorf("asset: assignment worker %d: %w", a.WorkerID, ErrForeignKeyViolation)
	}
	if a.EffectiveFrom.IsZero() {
		a.EffectiveFrom = time.Now().UTC()
	}
	// 复用既有落表(P2 早期 AssignAsset,无业务校验的纯 INSERT)。
	return s.AssignAsset(ctx, a)
}
