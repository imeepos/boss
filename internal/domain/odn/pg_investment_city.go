package odn

// 城市卷积(W5,审查 F1 ④):网格指标按 (prv,city) 向上卷积 + 分光容量户级口径。
// 与网格行同源(Go 层卷积 GridInvestment 结果,逐格一致可机械断言);
// 容量覆盖层只取城市域分光设备(odn_device prv/city 非空),导入域设备无城市维度,
// 只进 SplitCapacity 全网汇总(顺序五 W3 收口接通网格/设施后再入网格行)。

import (
	"context"
	"fmt"
)

// capacityByCitySQL 城市域分光设备容量卷积(fields.md 1.5.11 口径 8):
// 潜在户数=Σ二级容量+Σ无二级链的一级容量;已接=Σ已用端口占用。
var capacityByCitySQL = `
SELECT d.prv_code, d.city_prefix,
	COALESCE(SUM(sc.ratio) FILTER (WHERE sc.split_level = 2), 0)
		+ COALESCE(SUM(sc.ratio) FILTER (WHERE sc.split_level = 1 AND NOT sc.has_secondary), 0),
	COALESCE(SUM(sc.used_ports) FILTER (WHERE sc.split_level = 2), 0)
		+ COALESCE(SUM(sc.used_ports) FILTER (WHERE sc.split_level = 1 AND NOT sc.has_secondary), 0)
FROM odn_device_split_capacity sc
JOIN odn_device d ON d.id = sc.device_id
WHERE d.prv_code IS NOT NULL AND d.city_prefix IS NOT NULL
GROUP BY d.prv_code, d.city_prefix`

// CityInvestment 城市卷积列表:网格行卷积 + 容量列;容量表未部署(000221 代差)→ 容量列 nil。
func (s *PGStore) CityInvestment(ctx context.Context) ([]CityInvestmentRow, error) {
	gridRows, err := s.GridInvestment(ctx)
	if err != nil {
		return nil, err
	}
	homes, hasHomes, err := s.splitHomesByCity(ctx)
	if err != nil {
		return nil, err
	}
	cities := map[cityKey]*CityInvestmentRow{}
	for i := range gridRows {
		g := gridRows[i]
		k := cityKey{g.PrvCode, g.CityPrefix}
		c, ok := cities[k]
		if !ok {
			c = &CityInvestmentRow{PrvCode: g.PrvCode, CityPrefix: g.CityPrefix}
			cities[k] = c
		}
		c.GridCount++
		c.FacilitiesPlanned += g.FacilitiesPlanned
		c.FacilitiesInBuild += g.FacilitiesInBuild
		c.FacilitiesInService += g.FacilitiesInService
		c.FacilitiesRetired += g.FacilitiesRetired
		c.CoverageServed += g.CoverageServed
		c.CoveragePending += g.CoveragePending
		c.CoverageUnserved += g.CoverageUnserved
		c.SettledCost = addPtr(c.SettledCost, g.SettledCost)
		c.PlannedCost = addPtr(c.PlannedCost, g.PlannedCost)
		c.MaterialCost = addPtr(c.MaterialCost, g.MaterialCost)
	}
	out := []CityInvestmentRow{}
	for k, c := range cities {
		c.CostPerServed = costPerServed(c.SettledCost, c.CoverageServed)
		if hasHomes {
			if h, ok := homes[k]; ok {
				c.PotentialHomes = &h.potential
				c.ConnectedHomes = &h.connected
				exp := h.potential - h.connected
				c.ExpandableHomes = &exp
				c.CostPerPotential = costPerHomes(sumCostPtr(c.SettledCost, c.PlannedCost, c.MaterialCost), &h.potential)
			}
		}
		out = append(out, *c)
	}
	// 容量-only 城市(有城市域分光设备但无备案网格)补行,防容量静默丢失(网格列全 0,成本未登记)。
	if hasHomes {
		for k, h := range homes {
			if _, ok := cities[k]; ok {
				continue
			}
			c := &CityInvestmentRow{PrvCode: k.prvCode, CityPrefix: k.cityPrefix}
			c.PotentialHomes = &h.potential
			c.ConnectedHomes = &h.connected
			exp := h.potential - h.connected
			c.ExpandableHomes = &exp
			out = append(out, *c)
		}
	}
	sortCityRows(out)
	return out, nil
}

// cityHomes 城市户级容量(potential/connected)。
type cityHomes struct {
	potential int
	connected int
}

// splitHomesByCity 容量卷积;表未建(000221 未部署)→ hasHomes=false,容量列显式 nil。
func (s *PGStore) splitHomesByCity(ctx context.Context) (map[cityKey]cityHomes, bool, error) {
	ok, err := s.hasTable(ctx, "odn_device_split_capacity")
	if err != nil || !ok {
		return nil, false, err
	}
	rows, err := s.db.Query(ctx, capacityByCitySQL)
	if err != nil {
		return nil, false, fmt.Errorf("odn: capacity by city query: %w", err)
	}
	defer rows.Close()
	out := map[cityKey]cityHomes{}
	for rows.Next() {
		var k cityKey
		var h cityHomes
		if err := rows.Scan(&k.prvCode, &k.cityPrefix, &h.potential, &h.connected); err != nil {
			return nil, false, fmt.Errorf("odn: capacity by city scan: %w", err)
		}
		out[k] = h
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("odn: capacity by city rows: %w", err)
	}
	return out, true, nil
}

// addPtr 成本列卷积:双侧 nil→nil;单侧 nil 按 0 计入(城市内部分网格已登记)。
func addPtr(a, b *float64) *float64 {
	if a == nil && b == nil {
		return nil
	}
	v := 0.0
	if a != nil {
		v += *a
	}
	if b != nil {
		v += *b
	}
	return &v
}

// sortCityRows 城市行稳定排序(prv,city),避免 map 卷积顺序抖动。
func sortCityRows(rows []CityInvestmentRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a, b := rows[j-1], rows[j]
			if a.PrvCode < b.PrvCode || (a.PrvCode == b.PrvCode && a.CityPrefix <= b.CityPrefix) {
				break
			}
			rows[j-1], rows[j] = rows[j], rows[j-1]
		}
	}
}
