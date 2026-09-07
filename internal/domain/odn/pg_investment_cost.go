package odn

// 投资测算成本扩展(W5,审查 F1 ①②):规划成本与材料成本按项目明细金额占比分摊到网格。
// 分摊规则(fields.md 1.5.11 口径 5/6):项目级总额(预算/材料合计)无行级网格维度,
// 占比=网格内明细金额÷项目有网格归属明细合计;零归属明细的项目不摊(显式未登记)。
// 材料成本只计 CONFIRMED 出库单;资产单价经 采购入库链(assets.batch_id→procurement_receipts
// →procurement_order_items)取同单同料 id 最小一行;无采购价链资产不计额(口径注释)。

import (
	"context"
	"fmt"
)

// plannedCostByGridSQL 规划成本:项目预算(budget_amount,000218)×明细金额占比。
var plannedCostByGridSQL = `
WITH item_grid AS (
	SELECT ci.project_id, fl.prv_code, fl.city_prefix, fl.grid_code, SUM(ci.amount) AS part
	FROM construction_items ci
	JOIN odn_facility fl ON fl.code = ci.facility_code AND fl.grid_code IS NOT NULL
	GROUP BY ci.project_id, fl.prv_code, fl.city_prefix, fl.grid_code
), proj_total AS (
	SELECT project_id, SUM(part) AS total FROM item_grid GROUP BY project_id
)
SELECT i.prv_code, i.city_prefix, i.grid_code, SUM(cp.budget_amount * i.part / p.total)
FROM item_grid i
JOIN proj_total p ON p.project_id = i.project_id
JOIN construction_projects cp ON cp.id = i.project_id
WHERE cp.budget_amount IS NOT NULL AND p.total > 0
GROUP BY i.prv_code, i.city_prefix, i.grid_code`

// materialCostByGridSQL 材料成本:CONFIRMED 出库单资产采购价合计×明细金额占比。
var materialCostByGridSQL = `
WITH asset_price AS (
	SELECT DISTINCT ON (a.id) a.id, oi.unit_amount
	FROM assets a
	JOIN procurement_receipts r ON r.batch_id = a.batch_id
	JOIN procurement_order_items oi ON oi.order_id = r.order_id AND oi.material_code = a.type
	ORDER BY a.id, oi.id
), issue_cost AS (
	SELECT mi.project_id, SUM(p.unit_amount) AS cost
	FROM odn_material_issues mi
	JOIN odn_material_issue_items it ON it.issue_id = mi.id
	JOIN asset_price p ON p.id = it.asset_id
	WHERE mi.status = 'CONFIRMED'
	GROUP BY mi.project_id
), item_grid AS (
	SELECT ci.project_id, fl.prv_code, fl.city_prefix, fl.grid_code, SUM(ci.amount) AS part
	FROM construction_items ci
	JOIN odn_facility fl ON fl.code = ci.facility_code AND fl.grid_code IS NOT NULL
	GROUP BY ci.project_id, fl.prv_code, fl.city_prefix, fl.grid_code
), proj_total AS (
	SELECT project_id, SUM(part) AS total FROM item_grid GROUP BY project_id
)
SELECT i.prv_code, i.city_prefix, i.grid_code, SUM(ic.cost * i.part / p.total)
FROM item_grid i
JOIN proj_total p ON p.project_id = i.project_id
JOIN issue_cost ic ON ic.project_id = i.project_id
WHERE p.total > 0
GROUP BY i.prv_code, i.city_prefix, i.grid_code`

// plannedCosts 规划成本按网格;budget_amount 列未部署(W6 代差)→ hasData=false 显式未登记。
func (s *PGStore) plannedCosts(ctx context.Context) (map[gridKey]float64, bool, error) {
	ok, err := s.hasColumn(ctx, "construction_projects", "budget_amount")
	if err != nil || !ok {
		return nil, false, err
	}
	return s.queryGridCost(ctx, plannedCostByGridSQL, "planned cost")
}

// materialCosts 材料成本按网格;出库表未部署(W8 代差)→ hasData=false 显式未登记。
func (s *PGStore) materialCosts(ctx context.Context) (map[gridKey]float64, bool, error) {
	ok, err := s.hasTable(ctx, "odn_material_issues")
	if err != nil || !ok {
		return nil, false, err
	}
	return s.queryGridCost(ctx, materialCostByGridSQL, "material cost")
}

// queryGridCost 成本聚合 SQL 公共读法:键=网格复合主键,值=分摊金额。
func (s *PGStore) queryGridCost(ctx context.Context, sql, label string) (map[gridKey]float64, bool, error) {
	rows, err := s.db.Query(ctx, sql)
	if err != nil {
		return nil, false, fmt.Errorf("odn: %s query: %w", label, err)
	}
	defer rows.Close()
	out := map[gridKey]float64{}
	for rows.Next() {
		var k gridKey
		var amount float64
		if err := rows.Scan(&k.prvCode, &k.cityPrefix, &k.gridCode, &amount); err != nil {
			return nil, false, fmt.Errorf("odn: %s scan: %w", label, err)
		}
		out[k] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("odn: %s rows: %w", label, err)
	}
	return out, true, nil
}

// hasTable to_regclass 探测表是否存在(部署代差守卫,W2 settledCosts 先例)。
func (s *PGStore) hasTable(ctx context.Context, table string) (bool, error) {
	var reg any
	if err := s.db.QueryRow(ctx,
		"SELECT to_regclass($1)", "public."+table).Scan(&reg); err != nil {
		return false, fmt.Errorf("odn: probe table %s: %w", table, err)
	}
	return reg != nil, nil
}

// hasColumn information_schema 探测列是否存在(表在而列未部署的代差,如 budget_amount)。
func (s *PGStore) hasColumn(ctx context.Context, table, column string) (bool, error) {
	ok, err := s.hasTable(ctx, table)
	if err != nil || !ok {
		return false, err
	}
	var n int
	if err := s.db.QueryRow(ctx,
		"SELECT count(*) FROM information_schema.columns WHERE table_name=$1 AND column_name=$2",
		table, column).Scan(&n); err != nil {
		return false, fmt.Errorf("odn: probe column %s.%s: %w", table, column, err)
	}
	return n > 0, nil
}
