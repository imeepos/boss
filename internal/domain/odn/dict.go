package odn

// 省市编码字典只读视图(000075 种子;前端级联下拉数据源,契约 admin/odn.yaml)。

// RegionOption 省级选项(PRV 码 + 规范名)。
type RegionOption struct {
	PrvCode string `json:"prvCode"` // 如 PHL001
	Name    string `json:"name"`    // 规范名,如 国家首都区
}

// CityOption 城市前缀选项(省内唯一,规范 2.3)。
type CityOption struct {
	CityPrefix string `json:"cityPrefix"` // 如 MNL
	Name       string `json:"name"`
}
