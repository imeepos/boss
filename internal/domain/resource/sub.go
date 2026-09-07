package resource

import (
	"context"
	"time"
)

// Transfer 资源调拨单(设备跨区域调拨)。
type Transfer struct {
	ID              int64  `json:"id"`
	TransferNo      string `json:"transferNo"`
	ResourceID      int64  `json:"resourceId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	FromRegionID    int64  `json:"fromRegionId"`
	ToRegionID      int64  `json:"toRegionId"`
	Status          string `json:"status"` // PENDING/DOING/DONE
}

// Expansion 扩容单(公司对目标区域的端口扩容需求)。
type Expansion struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	ExpansionNo   string `json:"expansionNo"`
	RegionID      int64  `json:"regionId"`
	ExpectedPorts int32  `json:"expectedPorts"`
	Status        string `json:"status"` // PENDING/DOING/DONE
}

// ReserveRecord 端口预占记录(预占/释放流水)。
type ReserveRecord struct {
	ID      int64  `json:"id"`
	PortID  int64  `json:"portId"`
	OrderID int64  `json:"orderId"`
	Status  string `json:"status"` // HELD/RELEASED/CONSUMED
}

// PortChangeHistory 端口状态变更历史。
type PortChangeHistory struct {
	ID        int64     `json:"id"`
	PortID    int64     `json:"portId"`
	Status    string    `json:"status"`  // IDLE/RESERVED/USED/DISABLED
	OrderID   int64     `json:"orderId"` // 0=空
	ChangedAt time.Time `json:"changedAt"`
}

// ResourceSubService 网络资源子表域服务口(阶段4)。
type ResourceSubService interface {
	ListTransfers(ctx context.Context) ([]Transfer, error)
	CreateTransfer(ctx context.Context, t Transfer) (int64, error)
	ListExpansions(ctx context.Context) ([]Expansion, error)
	CreateExpansion(ctx context.Context, e Expansion) (int64, error)
	ListReserveRecords(ctx context.Context, portID int64) ([]ReserveRecord, error)
	AppendReserveRecord(ctx context.Context, r ReserveRecord) (int64, error)
	ListPortHistory(ctx context.Context, portID int64) ([]PortChangeHistory, error)
	AppendPortHistory(ctx context.Context, h PortChangeHistory) (int64, error)

	// ApproveTransfer/RejectTransfer 调拨审批(仅 PENDING 可审;驳回→DONE 终态)。
	ApproveTransfer(ctx context.Context, transferNo string) error
	RejectTransfer(ctx context.Context, transferNo string) error
	// ExecuteExpansion 执行扩容单(P5 补齐):目标设备按 expectedPorts 批量建端口(续号),
	// 全部就位→DONE;建口失败留 PENDING 可重试;设备须归属扩容单同一法人。
	ExecuteExpansion(ctx context.Context, expansionNo string, resourceID int64) (*ExpansionResult, error)
	// RejectExpansion 扩容驳回:PENDING→DONE 终态。
	RejectExpansion(ctx context.Context, expansionNo string) error
	// ReleaseReserve 手动释放预占(oss.yaml releaseReserve):回收端口 + 记录置 RELEASED。
	ReleaseReserve(ctx context.Context, reserveID int64) error
}
