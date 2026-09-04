package order

import (
	"context"
	"time"
)

// Dismantle 拆机单(回收资产释放端口)。
type Dismantle struct {
	ID              int64  `json:"id"`
	DismantleNo     string `json:"dismantleNo"`
	OrderID         int64  `json:"orderId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	AssetID         int64  `json:"assetId"`
	PortID          int64  `json:"portId"`
	Status          string `json:"status"` // PENDING/DOING/DONE/FAILED
}

// ActivationCallback 激活回调(订单第11环节的系统间回执)。
// TriedAt 最近尝试时刻(000182):同订单唯一行 upsert 随每次尝试刷新,lastTry 可见。
type ActivationCallback struct {
	ID      int64      `json:"id"`
	OrderID int64      `json:"orderId"`
	Result  string     `json:"result"` // SUCCESS/FAILED
	Retries int16      `json:"retries"`
	TriedAt *time.Time `json:"triedAt,omitempty"`
}

// DispatchTransfer 派单改派台账(工单师傅每次改派)。
type DispatchTransfer struct {
	ID                int64     `json:"id"`
	TicketID          int64     `json:"ticketId"`
	FromWorkerID      int64     `json:"fromWorkerId"` // 0=空
	FromWorkerName    string    `json:"fromWorkerName"`
	ToWorkerID        int64     `json:"toWorkerId"` // 0=空
	ToWorkerName      string    `json:"toWorkerName"`
	Reason            string    `json:"reason"`
	OperatorAccountID int64     `json:"operatorAccountId"` // 0=空
	TransferredAt     time.Time `json:"transferredAt"`
}

// OrderLedgerService 订单台账子表服务口(拆机/回调/改派)。
type OrderLedgerService interface {
	ListDismantles(ctx context.Context) ([]Dismantle, error)
	CreateDismantle(ctx context.Context, d Dismantle) (int64, error)
	ListActivationCallbacks(ctx context.Context) ([]ActivationCallback, error)
	AppendActivationCallback(ctx context.Context, c ActivationCallback) (int64, error)
	// LatestActivationCallback 取订单最近一次激活尝试(任务A-d lastTry);
	// 无记录返回 (nil, nil)。
	LatestActivationCallback(ctx context.Context, orderID int64) (*ActivationCallback, error)
	// RetryActivationCallback 回调重试:重放环节11 确认(幂等落账,FAILED 可转 SUCCESS)。
	RetryActivationCallback(ctx context.Context, id int64) error
	ListDispatchTransfers(ctx context.Context, ticketID int64) ([]DispatchTransfer, error)
	AppendDispatchTransfer(ctx context.Context, t DispatchTransfer) (int64, error)
}
