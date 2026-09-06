// 资产持有台账写侧(P2-W2-T1 G/H):领用开段/归还闭段。
// 台账模型:每次领用一行,effective_from→effective_to 时间段,历史不随当前值漂移
// (fields.md §8 归属台账通用模式)。
package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

// ErrAssignmentClosed 持有段已闭合,重复归还(P2-W2-T1 H,40900)。
var ErrAssignmentClosed = errors.New("asset: assignment already closed")

// ReturnAssignment 归还(P2-W2-T1 H):闭合持有段(effective_to=now);
// 守卫 UPDATE(WHERE effective_to IS NULL)防并发双闭,0 行回查区分
// 不存在(ErrNotFound)与已闭合(ErrAssignmentClosed,40900)。返回闭合时间供审计。
func (s *PGStore) ReturnAssignment(ctx context.Context, id int64) (*time.Time, error) {
	var effTo pgtype.Timestamptz
	err := s.db.QueryRow(ctx,
		`UPDATE asset_assignments SET effective_to = now()
		 WHERE id = $1 AND effective_to IS NULL
		 RETURNING effective_to`, id).Scan(&effTo)
	if err == nil {
		t := effTo.Time
		return &t, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("asset: return assignment %d: %w", id, err)
	}
	// 0 行:区分不存在与已闭合。
	var probe pgtype.Timestamptz
	err = s.db.QueryRow(ctx,
		`SELECT effective_to FROM asset_assignments WHERE id = $1`, id).Scan(&probe)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("asset: return assignment %d lookup: %w", id, err)
	}
	if probe.Valid {
		return nil, fmt.Errorf("asset: assignment %d closed at %s: %w", id, probe.Time.Format(time.RFC3339), ErrAssignmentClosed)
	}
	// effective_to 仍 NULL 却 UPDATE 0 行:并发窗口已被他请求闭合,同口径拒绝。
	return nil, fmt.Errorf("asset: assignment %d concurrently closed: %w", id, ErrAssignmentClosed)
}
