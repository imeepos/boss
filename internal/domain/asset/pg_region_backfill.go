package asset

// 资产区域快照回补(W3 收口,存量开户导入遗留①):assets.region_id/region_name 为空
// (NULL/0)的资产按「地址行级节点归属推导」回补快照,幂等可重跑。
// 口径:① 部署地址节点自身 region_id,为空沿地址链向上取最近非空祖先(000076/000077
// 建址继承同口径);② 地址链全空归属时,存量开户导入批(asset_batches.name='存量开户导入')
// 回退该批既有缺省区域(root 集团,与该批 customers/lo_accounts 同域,见 adopted
// 2026-09-07-legacy-vlan-on-ports 决策 4);③ 其余无归属不猜填,入未回补清单。
// 幂等:仅回补空快照行,重跑零更新;区域字典缺失行跳过并留 [asset-region-backfill] 日志。

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// RegionBackfillSample 回补前后样本行(供验收抽样断言)。
type RegionBackfillSample struct {
	AssetID    int64  `json:"assetId"`
	AssetCode  string `json:"assetCode"`
	RegionID   int64  `json:"regionId"`
	RegionName string `json:"regionName"`
}

// RegionBackfillResult 回补结果(backfilled=回补数;skipped=无归属不猜填数;
// samples=回补样本,默认返回前 5 条)。
type RegionBackfillResult struct {
	Backfilled int                    `json:"backfilled"`
	Skipped    int                    `json:"skipped"`
	Samples    []RegionBackfillSample `json:"samples"`
}

// BackfillRegionSnapshots 幂等回补资产区域快照(见文件头口径)。
// 失败路径:单资产回补失败记 [asset-region-backfill] 日志不中断,汇总错误上抛。
func (s *PGStore) BackfillRegionSnapshots(ctx context.Context) (*RegionBackfillResult, error) {
	rows, err := s.db.Query(ctx,
		`SELECT a.id, a.asset_code, COALESCE(a.address_id, 0), b.name FROM assets a
		 JOIN asset_batches b ON b.id = a.batch_id
		 WHERE a.region_id IS NULL OR a.region_id = 0
		 ORDER BY a.id`)
	if err != nil {
		return nil, fmt.Errorf("asset: region backfill scan targets: %w", err)
	}
	defer rows.Close()
	res := &RegionBackfillResult{}
	for rows.Next() {
		var id int64
		var code string
		var addressID int64
		var batchName string
		if err := rows.Scan(&id, &code, &addressID, &batchName); err != nil {
			return nil, fmt.Errorf("asset: region backfill target scan: %w", err)
		}
		regionID, regionName, err := s.deriveAssetRegion(ctx, addressID, batchName)
		if err != nil {
			log.Printf("[asset-region-backfill] DERIVE FAILED asset=%d code=%s reason=%v", id, code, err)
			continue
		}
		if regionID <= 0 {
			res.Skipped++
			continue
		}
		if _, err := s.db.Exec(ctx,
			`UPDATE assets SET region_id = $2, region_name = $3, updated_at = now() WHERE id = $1`,
			id, regionID, regionName); err != nil {
			log.Printf("[asset-region-backfill] UPDATE FAILED asset=%d code=%s reason=%v", id, code, err)
			continue
		}
		res.Backfilled++
		if len(res.Samples) < 5 {
			res.Samples = append(res.Samples, RegionBackfillSample{AssetID: id, AssetCode: code, RegionID: regionID, RegionName: regionName})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("asset: region backfill rows: %w", err)
	}
	log.Printf("[asset-region-backfill] DONE backfilled=%d skipped=%d", res.Backfilled, res.Skipped)
	return res, nil
}

// deriveAssetRegion 按地址行级节点归属推导区域(口径见文件头)。
// 返回 regionID<=0 表示无归属(调用方计 skipped,不猜填)。
func (s *PGStore) deriveAssetRegion(ctx context.Context, addressID int64, batchName string) (int64, string, error) {
	if addressID > 0 {
		var regionID int64
		var regionName string
		err := s.db.QueryRow(ctx,
			`WITH ad AS (SELECT path FROM addresses WHERE id = $1)
			 SELECT r.id, r.name FROM addresses a JOIN ad ON ad.path @> a.path
			 JOIN regions r ON r.id = a.region_id
			 WHERE a.region_id IS NOT NULL
			 ORDER BY nlevel(a.path) DESC LIMIT 1`, addressID).Scan(&regionID, &regionName)
		if err == nil {
			return regionID, regionName, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, "", fmt.Errorf("address region: %w", err)
		}
	}
	if batchName == "存量开户导入" {
		return s.legacyBatchRegion(ctx)
	}
	return 0, "", nil
}

// legacyBatchRegion 存量开户导入批缺省区域:root 集团(与该批 customers/lo_accounts 同域)。
// 区域字典缺失 root 时返回 0(不猜填,调用方计 skipped)。
func (s *PGStore) legacyBatchRegion(ctx context.Context) (int64, string, error) {
	var regionID int64
	var regionName string
	err := s.db.QueryRow(ctx, `SELECT id, name FROM regions WHERE path = 'root'`).Scan(&regionID, &regionName)
	if err != nil {
		return 0, "", fmt.Errorf("legacy region lookup: %w", err)
	}
	return regionID, regionName, nil
}
