package odn

// 网格投资测算读模型(P-INFRA-1 W2 基础 + W5 深化;纯只读聚合,零写路径)。
// 口径登记 fields.md 1.5.11:
//   - 设施数:odn_facility 网格归属行(grid_code 非空,即 P/MH),按 lifecycle_status 分组计数;
//   - 覆盖地址数:address_coverage 经服务设施(facility_code→odn_facility)归属网格;
//     仅挂核心设备或未挂目标的覆盖行不进网格行(设备无网格维度;UNSERVED 默认无目标);
//   - 已结算工程成本 SettledCost:读 W1 承包商结算数据;结算源未登记或该网格无结算数据时
//     SettledCost=nil,页面显示「未登记」,禁止显示 0;
//   - 每可装地址成本 CostPerServed:SettledCost ÷ CoverageServed;分母 0 或成本未登记时 nil。
//
// W5 深化(审查 F1 四点):
//   - 规划成本 PlannedCost:W6 项目预算(未开工也有信号)按项目明细金额占比分摊到网格
//     (项目级总额无行级网格维度);零归属明细的项目不摊(显式未登记,不硬摊);
//   - 材料成本 MaterialCost:W8 CONFIRMED 出库单逐台资产经采购价链取单价合计后同比例分摊;
//     与已结算人工成本分列,禁止混算;
//   - 容量口径(潜在/已接/可扩户数)住设备维度:设备无网格维度,只进城市行
//     (CityInvestment)与全网/设备视图(SplitCapacity),不进网格行。
type GridInvestmentRow struct {
	PrvCode    string `json:"prvCode"`
	CityPrefix string `json:"cityPrefix"`
	GridCode   int16  `json:"gridCode"`
	GridName   string `json:"gridName"`

	FacilitiesPlanned   int `json:"facilitiesPlanned"`   // 规划 PLANNED
	FacilitiesInBuild   int `json:"facilitiesInBuild"`   // 施工中 IN_BUILD
	FacilitiesInService int `json:"facilitiesInService"` // 在网 IN_SERVICE
	FacilitiesRetired   int `json:"facilitiesRetired"`   // 退役 RETIRED

	CoverageServed   int `json:"coverageServed"`   // 可装 SERVED
	CoveragePending  int `json:"coveragePending"`  // 规划在建 PENDING
	CoverageUnserved int `json:"coverageUnserved"` // 未覆盖 UNSERVED

	SettledCost   *float64 `json:"settledCost"`   // nil=未登记(禁止显示 0)
	PlannedCost   *float64 `json:"plannedCost"`   // nil=未登记(项目预算分摊,W5)
	MaterialCost  *float64 `json:"materialCost"`  // nil=未登记(出库采购价分摊,W5)
	CostPerServed *float64 `json:"costPerServed"` // nil=未登记(分母 0 或成本未登记)
}

// CityInvestmentRow 城市卷积行(W5 顺序④):网格行按 (prv,city) 向上卷积 + 容量列。
// 成本列为城市内网格行已登记值的合计(全未登记→nil);容量列只由城市域分光设备进
// (odn_device prv/city 非空),导入域设备只进 SplitCapacity 全网汇总。
type CityInvestmentRow struct {
	PrvCode    string `json:"prvCode"`
	CityPrefix string `json:"cityPrefix"`
	GridCount  int    `json:"gridCount"`

	FacilitiesPlanned   int `json:"facilitiesPlanned"`
	FacilitiesInBuild   int `json:"facilitiesInBuild"`
	FacilitiesInService int `json:"facilitiesInService"`
	FacilitiesRetired   int `json:"facilitiesRetired"`

	CoverageServed   int `json:"coverageServed"`
	CoveragePending  int `json:"coveragePending"`
	CoverageUnserved int `json:"coverageUnserved"`

	SettledCost   *float64 `json:"settledCost"`
	PlannedCost   *float64 `json:"plannedCost"`
	MaterialCost  *float64 `json:"materialCost"`
	CostPerServed *float64 `json:"costPerServed"`

	PotentialHomes   *int     `json:"potentialHomes"`   // 潜在户数(home-passed 口径) nil=无容量建模
	ConnectedHomes   *int     `json:"connectedHomes"`   // 已接户数=Σ已用端口占用
	ExpandableHomes  *int     `json:"expandableHomes"`  // 可扩户数=潜在-已接
	CostPerPotential *float64 `json:"costPerPotential"` // 全口径成本÷潜在户数(审查 F1:分母不是覆盖户数)
}

// SplitCapacityRow 设备分光容量行(W5 回写建模,fields.md 1.5.15)。
type SplitCapacityRow struct {
	DeviceID        int64   `json:"deviceId"`
	Code            string  `json:"code"`
	Kind            string  `json:"kind"`
	SplitLevel      int     `json:"splitLevel"` // 1=一级(OBD) 2=二级(SBD)
	Ratio           int     `json:"ratio"`      // 分光比分母=端口容量
	ChainRows       int     `json:"chainRows"`
	UsedPorts       int     `json:"usedPorts"`  // 已用端口占用(链行端口标签去重)
	Expandable      int     `json:"expandable"` // Ratio-UsedPorts,「还能接几户」
	PrvCode         *string `json:"prvCode"`    // nil=导入域设备(无城市,不进城市行)
	CityPrefix      *string `json:"cityPrefix"`
	LifecycleStatus string  `json:"lifecycleStatus"`
	HasSecondary    bool    `json:"hasSecondary"` // 一级器下挂二级链(端口不到户)
}

// SplitCapacitySummary 全网容量汇总(含导入域设备;城市行只算城市域)。
type SplitCapacitySummary struct {
	Devices         int `json:"devices"`
	PotentialHomes  int `json:"potentialHomes"`
	ConnectedHomes  int `json:"connectedHomes"`
	ExpandableHomes int `json:"expandableHomes"`
}

// SplitCapacityReport 容量读视图:设备清单 + 全网汇总。
type SplitCapacityReport struct {
	Items   []SplitCapacityRow   `json:"items"`
	Summary SplitCapacitySummary `json:"summary"`
}

// SplitBackfillResult 分光比回写建模结果(幂等全量重建,POST /odn/resource-chains/backfill-split)。
type SplitBackfillResult struct {
	DevicesModeled  int `json:"devicesModeled"`
	Level1          int `json:"level1"`
	Level2          int `json:"level2"`
	UnresolvedCodes int `json:"unresolvedCodes"` // 编码解析不到设备的链(日志留痕,不建模)
}

// gridKey 网格复合主键(prv+city+grid),结算成本映射的定位键。
type gridKey struct {
	prvCode    string
	cityPrefix string
	gridCode   int16
}

// cityKey 城市复合键(prv+city),容量卷积定位键。
type cityKey struct {
	prvCode    string
	cityPrefix string
}

// costPerServed 每可装地址成本:成本未登记或分母 0 返回 nil(未登记,禁止显示 0)。
func costPerServed(settled *float64, served int) *float64 {
	if settled == nil || served <= 0 {
		return nil
	}
	v := *settled / float64(served)
	return &v
}

// sumCostPtr 三项成本合计(全 nil→nil;部分 nil 按 0 计入,投资全口径=规划+材料+已结算)。
func sumCostPtr(a, b, c *float64) *float64 {
	if a == nil && b == nil && c == nil {
		return nil
	}
	v := 0.0
	for _, p := range []*float64{a, b, c} {
		if p != nil {
			v += *p
		}
	}
	return &v
}

// costPerHomes 全口径成本÷潜在户数(分母缺失或 0 → nil)。
func costPerHomes(total *float64, homes *int) *float64 {
	if total == nil || homes == nil || *homes <= 0 {
		return nil
	}
	v := *total / float64(*homes)
	return &v
}

// homesPotential 户级口径(fields.md 1.5.11 口径 8):一条二级分光端口=一户。
// 潜在=Σ二级分光器容量+Σ无二级链的一级分光器容量(直达户);已接=Σ已用端口占用。
func homesPotential(level int, hasSecondary bool, ratio, used int) (potential, connected int) {
	if level == 2 || (level == 1 && !hasSecondary) {
		return ratio, used
	}
	return 0, 0
}
