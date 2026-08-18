package order

import "context"

// DispatchTicket 派单工单(订单1:1,指派师傅)。
type DispatchTicket struct {
	TicketID        int64  `json:"ticketId"`
	TicketNo        string `json:"ticketNo"`
	OrderID         int64  `json:"orderId"`
	WorkerID        int64  `json:"workerId"` // 0=未派
	WorkerName      string `json:"workerName"`
	GroupID         int64  `json:"groupId"` // 0=无
	GroupName       string `json:"groupName"`
	RegionID        int64  `json:"regionId"` // 0=无
	RegionName      string `json:"regionName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	Status          string `json:"status"` // PENDING/DOING/DONE/CANCELED
}

// Complaint 报障工单(客服域,客户报障与处理)。
type Complaint struct {
	ID              int64
	TicketNo        string
	CustomerID      int64
	OrderID         int64 // 0=无关联订单
	LegalEntityID   int64
	LegalEntityName string
	Type            string
	Status          string // OPEN/PROCESSING/CLOSED
}

// ScanLog 扫码绑定记录(装维扫码与预绑定标签比对)。
type ScanLog struct {
	ID         int64
	OrderID    int64
	WorkerID   int64
	WorkerName string
	TagID      int64
	Result     string // MATCH/MISMATCH/OFFLINE_CACHED
}

// WorkOrderService 订单工单/报障/扫码域服务口(阶段5 子表)。
type WorkOrderService interface {
	ListDispatchTickets(ctx context.Context) ([]DispatchTicket, error)
	GetDispatchTicketByNo(ctx context.Context, ticketNo string) (*DispatchTicket, error)
	// AssignDispatchTicket 指派师傅(workerID/workerName 回填工单)。
	AssignDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error
	CreateDispatchTicket(ctx context.Context, t DispatchTicket) (int64, error)
	ListComplaints(ctx context.Context) ([]Complaint, error)
	CreateComplaint(ctx context.Context, c Complaint) (int64, error)
	// CloseComplaint 投诉办结(ticketNo 寻址,status→CLOSED)。
	CloseComplaint(ctx context.Context, ticketNo string) error
	ListScanLogs(ctx context.Context, orderID int64) ([]ScanLog, error)
	AppendScanLog(ctx context.Context, l ScanLog) (int64, error)
}
