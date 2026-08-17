package resource

import (
	"context"
	"time"
)

// Transfer 资源调拨单(设备跨区域调拨)。
type Transfer struct {
	ID              int64
	TransferNo      string
	ResourceID      int64
	LegalEntityID   int64
	LegalEntityName string
	FromRegionID    int64
	ToRegionID      int64
	Status          string // PENDING/DOING/DONE
}

// Expansion 扩容单(公司对目标区域的端口扩容需求)。
type Expansion struct {
	ID            int64
	LegalEntityID int64
	ExpansionNo   string
	RegionID      int64
	ExpectedPorts int32
	Status        string // PENDING/DOING/DONE
}

// ReserveRecord 端口预占记录(预占/释放流水)。
type ReserveRecord struct {
	ID      int64
	PortID  int64
	OrderID int64
	Status  string // HELD/RELEASED/CONSUMED
}

// PortChangeHistory 端口状态变更历史。
type PortChangeHistory struct {
	ID        int64
	PortID    int64
	Status    string // IDLE/RESERVED/USED/DISABLED
	OrderID   int64  // 0=空
	ChangedAt time.Time
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
}
