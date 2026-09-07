package odn

// 网格投资测算读模型(P-INFRA-1 W2;纯只读聚合,零写路径)。
// 口径登记 fields.md 1.5.11:
//   - 设施数:odn_facility 网格归属行(grid_code 非空,即 P/MH),按 lifecycle_status 分组计数;
//   - 覆盖地址数:address_coverage 经服务设施(facility_code→odn_facility)归属网格;
//     仅挂核心设备或未挂目标的覆盖行不进网格行(设备无网格维度;UNSERVED 默认无目标);
//   - 已结算工程成本:读 W1 承包商结算数据;结算源未登记或该网格无结算数据时
//     SettledCost=nil,页面显示「未登记」,禁止显示 0;
//   - 每可装地址成本:SettledCost ÷ CoverageServed;分母 0 或成本未登记时 nil。
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
	CostPerServed *float64 `json:"costPerServed"` // nil=未登记(分母 0 或成本未登记)
}

// gridKey 网格复合主键(prv+city+grid),结算成本映射的定位键。
type gridKey struct {
	prvCode    string
	cityPrefix string
	gridCode   int16
}

// costPerServed 每可装地址成本:成本未登记或分母 0 返回 nil(未登记,禁止显示 0)。
func costPerServed(settled *float64, served int) *float64 {
	if settled == nil || served <= 0 {
		return nil
	}
	v := *settled / float64(served)
	return &v
}
