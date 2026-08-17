package order

import (
	"context"
	"time"
)

// Dismantle 拆机单(回收资产释放端口)。
type Dismantle struct {
	ID              int64
	DismantleNo     string
	OrderID         int64
	LegalEntityID   int64
	LegalEntityName string
	AssetID         int64
	PortID          int64
	Status          string // PENDING/DOING/DONE/FAILED
}

// ActivationCallback 激活回调(订单第11环节的系统间回执)。
type ActivationCallback struct {
	ID      int64
	OrderID int64
	Result  string // SUCCESS/FAILED
	Retries int16
}

// DispatchTransfer 派单改派台账(工单师傅每次改派)。
type DispatchTransfer struct {
	ID                int64
	TicketID          int64
	FromWorkerID      int64 // 0=空
	FromWorkerName    string
	ToWorkerID        int64 // 0=空
	ToWorkerName      string
	Reason            string
	OperatorAccountID int64 // 0=空
	TransferredAt     time.Time
}

// OrderLedgerService 订单台账子表服务口(拆机/回调/改派)。
type OrderLedgerService interface {
	ListDismantles(ctx context.Context) ([]Dismantle, error)
	CreateDismantle(ctx context.Context, d Dismantle) (int64, error)
	ListActivationCallbacks(ctx context.Context) ([]ActivationCallback, error)
	AppendActivationCallback(ctx context.Context, c ActivationCallback) (int64, error)
	ListDispatchTransfers(ctx context.Context, ticketID int64) ([]DispatchTransfer, error)
	AppendDispatchTransfer(ctx context.Context, t DispatchTransfer) (int64, error)
}
