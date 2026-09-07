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
	out := []GridInvestmentRow{}
	for rows.Next() {
		var r GridInvestmentRow
		if err := rows.Scan(&r.PrvCode, &r.CityPrefix, &r.GridCode, &r.GridName,
			&r.FacilitiesPlanned, &r.FacilitiesInBuild, &r.FacilitiesInService, &r.FacilitiesRetired,
			&r.CoverageServed, &r.CoveragePending, &r.CoverageUnserved); err != nil {
			return nil, fmt.Errorf("odn: grid investment scan: %w", err)
		}
		if hasCost {
			if amount, ok := costs[gridKey{r.PrvCode, r.CityPrefix, r.GridCode}]; ok {
				r.SettledCost = &amount
				r.CostPerServed = costPerServed(&amount, r.CoverageServed)
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("odn: grid investment rows: %w", err)
	}
	return out, nil
}

// settledCosts 按网格聚合已结算工程成本(W1 承包商结算数据源)。
// W1(P-INFRA-1)尚未合并进 main:结算表不存在,返回 hasCost=false,
// 页面对全部网格显示「未登记」(禁止显示 0)。
// W2 合并前反向同步 main 拿到结算数据源后,此处替换为真实结算表聚合,
// 口径见 fields.md 1.5.11;替换前标记:W2-COST-SOURCE-PENDING。
func (s *PGStore) settledCosts(_ context.Context) (map[gridKey]float64, bool, error) {
	return nil, false, nil
}
