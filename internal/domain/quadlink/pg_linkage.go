package quadlink

// 资产状态联动接线(P1-T1,adopted note 2026-09-06-asset-tag-p1-wave):
// AssetStateSink 窄接口由 asset.PGStore 隐式实现;四码域不直触 assets 表。
// 同库强一致(R1 调研裁定):联动在调用方事务上执行,失败整单回滚阻断 MATCH/拆机,
// 并留 [quadlink] ASSET LINKAGE FAILED 可 grep 日志;巡检(pg_patrol)只作兜底不替代。

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/ymm-001/boss/internal/domain/asset"
)

// AssetStateSink 资产状态联动窄接口(装机 DEPLOYED/拆机 IN_STOCK,资产域实现)。
type AssetStateSink interface {
	MarkDeployed(ctx context.Context, ex asset.ExecQuerier, assetID, addressID, workerID int64, workerName string) error
	MarkReleased(ctx context.Context, ex asset.ExecQuerier, assetID int64) error
}

// linkAsset 装机联动:资产置 DEPLOYED+绑地址+轨迹行(在 ex 事务上)。未装配 sink 或链路无资产跳过。
func (s *PGStore) linkAsset(ctx context.Context, ex asset.ExecQuerier, assetID, addressID, workerID int64, workerName string) error {
	if s.assets == nil || assetID <= 0 {
		return nil
	}
	if err := s.assets.MarkDeployed(ctx, ex, assetID, addressID, workerID, workerName); err != nil {
		slog.ErrorContext(ctx, "[quadlink] ASSET LINKAGE FAILED",
			"op", "deploy", "asset_id", assetID, "address_id", addressID, "err", err)
		return fmt.Errorf("quadlink: asset deploy linkage: %w", err)
	}
	return nil
}

// releaseAsset 拆机联动:资产回 IN_STOCK+清地址+轨迹行(在 ex 事务上)。
func (s *PGStore) releaseAsset(ctx context.Context, ex asset.ExecQuerier, assetID int64) error {
	if s.assets == nil || assetID <= 0 {
		return nil
	}
	if err := s.assets.MarkReleased(ctx, ex, assetID); err != nil {
		slog.ErrorContext(ctx, "[quadlink] ASSET LINKAGE FAILED",
			"op", "release", "asset_id", assetID, "err", err)
		return fmt.Errorf("quadlink: asset release linkage: %w", err)
	}
	return nil
}

// UnbindRequireScan 拆机必扫码:不扫码直接拒;扫码与已关联资产 EPC 不一致拒;一致则四码解绑
// (UNLINKED)并联动资产回 IN_STOCK——链路翻转与资产联动同一事务(P1-T1 强一致)。
func (s *PGStore) UnbindRequireScan(ctx context.Context, orderID int64, scannedEPC string) error {
	if scannedEPC == "" {
		return ErrScanRequired
	}
	link, err := s.linkByOrderCustomer(ctx, orderID)
	if err != nil {
		return err
	}
	var epc string
	err = s.db.QueryRow(ctx, `SELECT epc_code FROM tags WHERE bound_asset_id = $1`, link.AssetID).Scan(&epc)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: asset %d 无标签", ErrScanMismatch, link.AssetID)
	}
	if err != nil {
		return fmt.Errorf("quadlink: unbind tag: %w", err)
	}
	if epc != scannedEPC {
		return ErrScanMismatch
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("quadlink: begin unbind: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE quad_links SET status = 'UNLINKED' WHERE id = $1`, link.ID); err != nil {
		return fmt.Errorf("quadlink: unbind: %w", err)
	}
	// 资产联动(P1-T1):拆机释放;失败整单回滚,链路保持原状可重试。
	if err := s.releaseAsset(ctx, tx, link.AssetID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("quadlink: commit unbind: %w", err)
	}
	return nil
}
