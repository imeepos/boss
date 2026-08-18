package gis

// 阶段8 GIS 域服务(派生聚合):八级下钻、资产实时详情。
// 口径:八级 = 市(1)/区(2)/街道(3)/小区(4)/楼栋(5)=addresses 层级,
//       弱电井(6)=OLT 设备、分光器(7)=SPLITTER 设备(resources 树)、端口(8)=ports。

import (
	"context"
	"time"
)

// Node 八级下钻中某一级的节点(含下一级数量统计)。
type Node struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Level int16  `json:"level"`
	Count int64  `json:"count"` // 下一级直属数量(叶级=0)
}

// ResourceDetail 资产实时详情(合并资源状态 + 最新指标 + 关联用户 + 端口占用)。
type ResourceDetail struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Status       string     `json:"status"`
	AddressID    int64      `json:"addressId"`
	OpticalPower *float64   `json:"opticalPower"` // nil=无数据
	PacketLoss   *float64   `json:"packetLoss"`
	CustomerName string     `json:"customerName"` // 关联用户(经端口→四码→客户)
	PortsUsed    int64      `json:"portsUsed"`    // 占用端口数
	PortsTotal   int64      `json:"portsTotal"`
	CollectedAt  *time.Time `json:"collectedAt"`
}

// GISService GIS 域服务口(阶段8)。
type GISService interface {
	// Drill 八级下钻:返回指定 level 在 parentID 下的节点(parentID=0 表示顶层)。
	Drill(ctx context.Context, level int16, parentID int64) ([]Node, error)
	// LevelCounts 全量八级数量统计(大屏/首屏)。
	LevelCounts(ctx context.Context) ([]Node, error)
	// ResourceDetail 资产实时详情。
	ResourceDetail(ctx context.Context, resourceID int64) (*ResourceDetail, error)
}
