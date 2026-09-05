package asset

// 资产状态联动实现(P1-T1,adopted note 2026-09-06-asset-tag-p1-wave):
// quadlink 扫码/拆机经由窄接口在本域事务上原子完成"状态翻转+轨迹落行"。
// 同库强一致(R1 调研裁定):联动失败回滚阻断扫码,禁 best-effort 静默漂移;
// SELECT FOR UPDATE 行锁复检防并发(Snipe-IT checkout 同款)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ExecQuerier 事务面最小执行器(pgx.Tx 与 *pgxpool.Pool 天然满足),
// 联动方把事务传入,资产域不再自开事务。
type ExecQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ErrAssetScrapped 资产已报废(终态):部署/释放联动拒绝自动执行,转人工处理。
var ErrAssetScrapped = errors.New("asset: scrapped asset rejects state linkage")

// MarkDeployed 装机联动:IN_STOCK/MAINTENANCE → DEPLOYED 并绑地址,同事务落轨迹行。
// DEPLOYED 幂等 no-op(重放自愈);SCRAPPED 拒绝(转人工);地址名快照自查 addresses。
func (s *PGStore) MarkDeployed(ctx context.Context, ex ExecQuerier, assetID, addressID, workerID int64, workerName string) error {
	var status string
	err := ex.QueryRow(ctx, `SELECT status FROM assets WHERE id = $1 FOR UPDATE`, assetID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: mark deployed lock: %w", err)
	}
	if status == "DEPLOYED" {
		return nil
	}
	if status == "SCRAPPED" {
		return ErrAssetScrapped
	}
	if _, err := ex.Exec(ctx,
		`UPDATE assets SET status = 'DEPLOYED', address_id = $2, updated_at = now() WHERE id = $1`,
		assetID, idOrNil(addressID)); err != nil {
		return fmt.Errorf("asset: mark deployed: %w", err)
	}
	addrName := ""
	if addressID > 0 {
		if err := ex.QueryRow(ctx, `SELECT COALESCE(name, '') FROM addresses WHERE id = $1`, addressID).Scan(&addrName); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("asset: mark deployed address: %w", err)
		}
	}
	if _, err := ex.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, address_id, address_name, worker_id, worker_name, changed_at)
		 VALUES($1, 'DEPLOYED', $2, $3, $4, $5, now())`,
		assetID, idOrNil(addressID), addrName, idOrNil(workerID), workerName); err != nil {
		return fmt.Errorf("asset: mark deployed lifecycle: %w", err)
	}
	return nil
}

// MarkReleased 拆机联动:DEPLOYED → IN_STOCK 并清地址(同一 UPDATE 原子,防"DEPLOYED 但地址空"),
// 同事务落轨迹行。IN_STOCK 幂等 no-op;SCRAPPED 拒绝(报废件不回库存,转人工)。
func (s *PGStore) MarkReleased(ctx context.Context, ex ExecQuerier, assetID int64) error {
	var status string
	err := ex.QueryRow(ctx, `SELECT status FROM assets WHERE id = $1 FOR UPDATE`, assetID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: mark released lock: %w", err)
	}
	if status == "IN_STOCK" {
		return nil
	}
	if status == "SCRAPPED" {
		return ErrAssetScrapped
	}
	if _, err := ex.Exec(ctx,
		`UPDATE assets SET status = 'IN_STOCK', address_id = NULL, updated_at = now() WHERE id = $1`,
		assetID); err != nil {
		return fmt.Errorf("asset: mark released: %w", err)
	}
	if _, err := ex.Exec(ctx,
		`INSERT INTO asset_lifecycles(asset_id, status, changed_at) VALUES($1, 'IN_STOCK', now())`,
		assetID); err != nil {
		return fmt.Errorf("asset: mark released lifecycle: %w", err)
	}
	return nil
}
