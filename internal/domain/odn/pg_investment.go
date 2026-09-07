package odn

import (
	"context"
	"fmt"
)

// gridInvestmentSQL 网格全量行左联设施生命周期计数与覆盖状态计数(单一只读 SQL)。
// 设施口径:grid_code 非空行(P/MH);覆盖口径:经服务设施归属网格(fields.md 1.5.11)。
var gridInvestmentSQL = `
SELECT g.prv_code, g.city_prefix, g.grid_code, COALESCE(g.name, ''),
	COALESCE(f.planned, 0), COALESCE(f.in_build, 0), COALESCE(f.in_service, 0), COALESCE(f.retired, 0),
	COALESCE(c.served, 0), COALESCE(c.pending, 0), COALESCE(c.unserved, 0)
FROM odn_grid g
LEFT JOIN (
	SELECT prv_code, city_prefix, grid_code,
		COUNT(*) FILTER (WHERE lifecycle_status = 'PLANNED') AS planned,
		COUNT(*) FILTER (WHERE lifecycle_status = 'IN_BUILD') AS in_build,
		COUNT(*) FILTER (WHERE lifecycle_status = 'IN_SERVICE') AS in_service,
		COUNT(*) FILTER (WHERE lifecycle_status = 'RETIRED') AS retired
	FROM odn_facility
	WHERE grid_code IS NOT NULL
	GROUP BY prv_code, city_prefix, grid_code
) f ON f.prv_code = g.prv_code AND f.city_prefix = g.city_prefix AND f.grid_code = g.grid_code
LEFT JOIN (
	SELECT fl.prv_code, fl.city_prefix, fl.grid_code,
		COUNT(*) FILTER (WHERE cv.status = 'SERVED') AS served,
		COUNT(*) FILTER (WHERE cv.status = 'PENDING') AS pending,
		COUNT(*) FILTER (WHERE cv.status = 'UNSERVED') AS unserved
	FROM address_coverage cv
	JOIN odn_facility fl ON fl.code = cv.facility_code AND fl.grid_code IS NOT NULL
	GROUP BY fl.prv_code, fl.city_prefix, fl.grid_code
) c ON c.prv_code = g.prv_code AND c.city_prefix = g.city_prefix AND c.grid_code = g.grid_code
ORDER BY g.prv_code, g.city_prefix, g.grid_code
`

// GridInvestment 网格投资测算聚合:网格全量 + 生命周期/覆盖计数 + 已结算成本。
// 纯只读:单连接两次查询,无任何写入。
func (s *PGStore) GridInvestment(ctx context.Context) ([]GridInvestmentRow, error) {
	rows, err := s.db.Query(ctx, gridInvestmentSQL)
	if err != nil {
		return nil, fmt.Errorf("odn: grid investment query: %w", err)
	}
	defer rows.Close()

	costs, hasCost, err := s.settledCosts(ctx)
	if err != nil {
		return nil, err
	}
	planned, hasPlanned, err := s.plannedCosts(ctx)
	if err != nil {
		return nil, err
	}
	material, hasMaterial, err := s.materialCosts(ctx)
	if err != nil {
		return nil, err
	}
	out := []GridInvestmentRow{}
	for rows.Next() {
		var r GridInvestmentRow
		if err := rows.Scan(&r.PrvCode, &r.CityPrefix, &r.GridCode, &r.GridName,
			&r.FacilitiesPlanned, &r.FacilitiesInBuild, &r.FacilitiesInService, &r.FacilitiesRetired,
			&r.CoverageServed, &r.CoveragePending, &r.CoverageUnserved); err != nil {
			return nil, fmt.Errorf("odn: grid investment scan: %w", err)
		}
		k := gridKey{r.PrvCode, r.CityPrefix, r.GridCode}
		if hasCost {
			if amount, ok := costs[k]; ok {
				r.SettledCost = &amount
				r.CostPerServed = costPerServed(&amount, r.CoverageServed)
			}
		}
		if hasPlanned {
			if amount, ok := planned[k]; ok {
				r.PlannedCost = &amount
			}
		}
		if hasMaterial {
			if amount, ok := material[k]; ok {
				r.MaterialCost = &amount
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("odn: grid investment rows: %w", err)
	}
	return out, nil
}

// settledCostByGridSQL 已结算工程成本按网格归集(W1 工程结算数据源,迁移 000206):
// 结算单 SETTLED(已结算;PENDING 未结算/VOIDED 作废不计)→ 项目明细设施归属网格
// → 汇总明细金额(ACCEPTED 后明细锁定,项目应付总额=明细金额和,不漂移)。
var settledCostByGridSQL = `SELECT fl.prv_code, fl.city_prefix, fl.grid_code, SUM(ci.amount)
	FROM construction_settlements cs
	JOIN construction_projects cp ON cp.id = cs.project_id
	JOIN construction_items ci ON ci.project_id = cp.id
	JOIN odn_facility fl ON fl.code = ci.facility_code AND fl.grid_code IS NOT NULL
	WHERE cs.status = 'SETTLED'
	GROUP BY fl.prv_code, fl.city_prefix, fl.grid_code`

// settledCosts 按网格聚合已结算工程成本(W1 承包商结算数据源,迁移 000206)。
// 结算表未建(W1 尚未合并/102 未部署该迁移)时返回 hasCost=false,页面显示
// 「未登记」,禁止显示 0;表存在而查询失败按错误上抛,禁止静默吞错。
// 口径:cs.status=SETTLED;明细经 facility_code 归网格,无网格维度设施
// (TW/CLS/TBX)的明细金额不进网格行,详见 fields.md 1.5.11。
func (s *PGStore) settledCosts(ctx context.Context) (map[gridKey]float64, bool, error) {
	var reg any
	if err := s.db.QueryRow(ctx, `SELECT to_regclass('construction_settlements')`).Scan(&reg); err != nil {
		return nil, false, fmt.Errorf("odn: settled cost probe: %w", err)
	}
	if reg == nil {
		return nil, false, nil // W1 结算源未登记:显式未登记而非 0
	}
	rows, err := s.db.Query(ctx, settledCostByGridSQL)
	if err != nil {
		return nil, false, fmt.Errorf("odn: settled cost query: %w", err)
	}
	defer rows.Close()
	out := map[gridKey]float64{}
	for rows.Next() {
		var k gridKey
		var amount float64
		if err := rows.Scan(&k.prvCode, &k.cityPrefix, &k.gridCode, &amount); err != nil {
			return nil, false, fmt.Errorf("odn: settled cost scan: %w", err)
		}
		out[k] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("odn: settled cost rows: %w", err)
	}
	return out, true, nil
}
