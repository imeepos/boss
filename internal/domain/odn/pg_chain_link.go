package odn

// W3 收口(F2,审查 2026-09-07-invest-build-phase-design-review):资源链导入网格归属联动。
// 链行可选携带城市/网格归属(prv/city/grid 三列一体,校验见 validateChainAttribution),
// 带归属的链行导入时:①备案网格(odn_grid)②建/复用网格锚点设施(odn_facility,kind=MH)
// ——投资测算(/odn/grid-investment 网格行设施计数)与覆盖登记(挂锚点设施码进覆盖页)
// 两链路数据面由此接通;不携带归属的链行走 000210 原逻辑,零行为变化。
// 失败路径:行级失败随事务回滚并留 [odn-import] 可 grep 日志,禁止静默吞错。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ensureChainGrid 网格归属联动①:城市字典校验 + 网格幂等备案。
// (prv,city) 须在 odn_city_code 已登记(FK 权威),未登记拒绝该行(不猜填);
// 既有网格 DO NOTHING 不动(零改动),名称缺省=机房名称或 "导入网格-<grid>"。
func ensureChainGrid(ctx context.Context, tx pgx.Tx, rec *ChainRecord) error {
	if rec.GridCode == 0 {
		return nil
	}
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM odn_city_code WHERE prv_code=$1 AND city_prefix=$2)`,
		rec.PrvCode, rec.CityPrefix).Scan(&exists); err != nil {
		return fmt.Errorf("odn: probe city: %w", err)
	}
	if !exists {
		return fmt.Errorf("odn: 城市未登记: %s/%s", rec.PrvCode, rec.CityPrefix)
	}
	name := rec.SiteName
	if name == "" {
		name = fmt.Sprintf("导入网格-%02d", rec.GridCode)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO odn_grid (prv_code, city_prefix, grid_code, name)
		 VALUES ($1,$2,$3,$4) ON CONFLICT (prv_code, city_prefix, grid_code) DO NOTHING`,
		rec.PrvCode, rec.CityPrefix, rec.GridCode, name); err != nil {
		return fmt.Errorf("odn: upsert grid: %w", err)
	}
	return nil
}

// expandChainAnchor 网格归属联动②:网格内建/复用一条 kind=MH 锚点设施。
// 编码=MH+2位网格码+3位序号(规范 4.1 网格分区模式,编码即网格归属);
// 名称=机房名称优先,否则 "链锚-<首箱体码>"(同链重导/同批多行复用同一锚点);
// lifecycle 与链行同步(规划链落 PLANNED,投资测算网格行设施计数非空)。
// 覆盖登记(POST /odn/coverage)挂锚点设施码即进入该网格行覆盖计数与覆盖页。
func expandChainAnchor(ctx context.Context, tx pgx.Tx, rec *ChainRecord) error {
	if rec.GridCode == 0 {
		return nil
	}
	name := chainAnchorName(rec)
	var code string
	err := tx.QueryRow(ctx,
		`SELECT code FROM odn_facility
		 WHERE prv_code=$1 AND city_prefix=$2 AND grid_code=$3 AND kind='MH'
		   AND name=$4 AND status <> 'RETIRED'
		 ORDER BY code LIMIT 1`, rec.PrvCode, rec.CityPrefix, rec.GridCode, name).Scan(&code)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("odn: lookup chain anchor: %w", err)
	}
	code, err = nextFacilityCodeTx(ctx, tx, KindManhole, int16(rec.GridCode))
	if err != nil {
		return fmt.Errorf("odn: anchor code: %w", err)
	}
	// 并发导入同链同网格时编码冲突:DO NOTHING 后重查,幂等复用(不整行失败)。
	if _, err := tx.Exec(ctx,
		`INSERT INTO odn_facility
		 (code, kind, prv_code, city_prefix, grid_code, name, status, lifecycle_status)
		 VALUES ($1,$2,$3,$4,$5,$6,'IN_USE',$7)
		 ON CONFLICT (code) DO NOTHING`,
		code, KindManhole, rec.PrvCode, rec.CityPrefix, rec.GridCode, name, rec.Lifecycle); err != nil {
		return fmt.Errorf("odn: insert chain anchor: %w", err)
	}
	// 冲突时由先提交事务写入,重查确认存在即复用。
	var got string
	if err := tx.QueryRow(ctx,
		`SELECT code FROM odn_facility WHERE prv_code=$1 AND city_prefix=$2 AND grid_code=$3
		   AND kind='MH' AND name=$4 AND status <> 'RETIRED' ORDER BY code LIMIT 1`,
		rec.PrvCode, rec.CityPrefix, rec.GridCode, name).Scan(&got); err != nil {
		return fmt.Errorf("odn: reload chain anchor: %w", err)
	}
	return nil
}

// chainAnchorName 锚点设施名:机房名称优先,空则链首箱体码(OCC/ODF/OLT)。
func chainAnchorName(rec *ChainRecord) string {
	if rec.SiteName != "" {
		return rec.SiteName
	}
	for _, c := range []string{rec.OccCode, rec.OdfCode, rec.OltCode} {
		if c != "" {
			return "链锚-" + c
		}
	}
	return "链锚"
}

// nextFacilityCodeTx 事务内下一设施编码(与 NextFacilityCode 同口径:现存 MAX+1,
// 序号段按 facilitySeq.offset/digits 精确切片,共用 nextFacilityCode 免口径分叉)。
func nextFacilityCodeTx(ctx context.Context, tx pgx.Tx, kind string, gridCode int16) (string, error) {
	return nextFacilityCode(ctx, tx, kind, gridCode)
}
