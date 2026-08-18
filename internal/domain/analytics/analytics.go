package analytics

// Package analytics 阶段9:经营分析域(派生聚合,无基表;口径公开可解释)。
// 五大指标、投资一张图热力图、维护一张表。

import (
	"context"
)

// Indicator 单个核心指标(含口径说明,前端直出可解释)。
type Indicator struct {
	Key    string  `json:"key"`    // portUtilization/installConversion/maintenanceCostPerUser/assetHealth/regionROI
	Name   string  `json:"name"`   // 中文名
	Value  float64 `json:"value"`  // 比率类 0~1;成本为元;评分为 0~100
	Detail string  `json:"detail"` // 口径与分子/分母明细
}

// RegionROI 区域投资回报率明细(支持下钻)。
type RegionROI struct {
	RegionID   int64   `json:"regionId"`
	RegionName string  `json:"regionName"`
	Revenue    float64 `json:"revenue"`    // 已收款(payments SUCCESS)
	Investment float64 `json:"investment"` // 扩容端口×单端口成本
	ROI        float64 `json:"roi"`        // Investment>0 时 =Revenue/Investment,否则 0
}

// HeatCell 投资一张图热力格子(小区/楼栋粒度)。
type HeatCell struct {
	AddressID   int64   `json:"addressId"`
	Name        string  `json:"name"`
	Level       int16   `json:"level"` // 4 小区 / 5 楼栋
	PortsTotal  int64   `json:"portsTotal"`
	PortsUsed   int64   `json:"portsUsed"`
	Utilization float64 `json:"utilization"` // 0~1
}

// MaintenanceItem 维护一张表行(排序可解释)。
type MaintenanceItem struct {
	DeviceNo    string  `json:"deviceNo"`
	DeviceType  string  `json:"deviceType"`
	HealthScore int16   `json:"healthScore"`
	FaultCount  int32   `json:"faultCount"`
	AgeYears    float64 `json:"ageYears"`
	Reason      string  `json:"reason"`
	Priority    string  `json:"priority"`
}

// AnalyticsService 阶段9 分析域口。
type AnalyticsService interface {
	// FiveIndicators 五大核心指标 + 区域 ROI 明细(可下钻)。
	FiveIndicators(ctx context.Context) ([]Indicator, []RegionROI, error)
	// Heatmap 投资一张图:按地址(level 4 小区/5 楼栋)聚合端口利用率。
	Heatmap(ctx context.Context) ([]HeatCell, error)
	// MaintenanceList 维护一张表:优先级→健康度→故障率→年限排序。
	MaintenanceList(ctx context.Context) ([]MaintenanceItem, error)
}
