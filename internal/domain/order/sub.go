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
	ID              int64  `json:"id"`
	TicketNo        string `json:"ticketNo"`
	CustomerID      int64  `json:"customerId"`
	OrderID         int64  `json:"orderId"` // 0=无关联订单
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	Type            string `json:"type"`
	Status          string `json:"status"` // OPEN/PROCESSING/CLOSED
}

// ScanLog 扫码绑定记录(装维扫码与预绑定标签比对)。
type ScanLog struct {
	ID         int64  `json:"id"`
	OrderID    int64  `json:"orderId"`
	WorkerID   int64  `json:"workerId"`
	WorkerName string `json:"workerName"`
	TagID      int64  `json:"tagId"`
	Result     string `json:"result"` // MATCH/MISMATCH/OFFLINE_CACHED
}

// TicketItem 派单工单列表读模型:联表补订单环节/客户/安装地址(师傅端列表视图)。
//
// 详情视图(api/openapi/worker/schemas.yaml::TicketDetail)需要更多字段:
//   CustomerPhone/OfferName/SplitterPort/PreBindTag/ScheduleSlot
//   /FaultTypeLabel/ReportedAt/SlaLeftMinutes/RemoteDiagnosis/FinishedAt。
// 实际有数据支撑的:CustomerPhone、OfferName、FinishedAt(由 MAX(order_stages.finished_at)
// 在 status=DONE 时派生)。其余字段是 OpenAPI 预留位,后端暂未提供数据源,返回空串/0,
// 前端按"空值不渲染"处理。
type TicketItem struct {
	TicketID     int64  `json:"ticketId"`
	TicketNo     string `json:"ticketNo"`
	OrderID      int64  `json:"orderId"`
	WorkerID     int64  `json:"workerId"`
	Status       string `json:"status"`      // PENDING/DOING/DONE/CANCELED
	OrderStatus  string `json:"orderStatus"` // 订单终态兜底:DONE/CANCELED 时工单视图按完成处理
	CustomerName string `json:"customerName"`
	CustomerPhone string `json:"customerPhone"` // 明文,handler 出门必须 httpx.MaskPhone 脱敏
	OfferName    string `json:"offerName"`      // → product_offers.name(下单快照)
	Address      string `json:"address"`
	Stage        int8   `json:"stage"` // 订单当前环节 1~12
	FinishedAt   string `json:"finishedAt"` // 订单 DONE 时 MAX(order_stages.finished_at) 派生,空=进行中

	// OpenAPI TicketDetail 预留位:后端尚未提供数据源,前端按空值不渲染处理。
	SplitterPort    string `json:"splitterPort"`
	PreBindTag      string `json:"preBindTag"`
	ScheduleSlot    string `json:"scheduleSlot"`
	FaultTypeLabel  string `json:"faultTypeLabel"`
	ReportedAt      string `json:"reportedAt"`
	SlaLeftMinutes  int    `json:"slaLeftMinutes"`  // 报障单 SLA 倒计时,后端未实现返回 0
	RemoteDiagnosis string `json:"remoteDiagnosis"` // 报障单远程诊断结论,后端未实现返回空
}

// WorkOrderService 订单工单/报障/扫码域服务口(阶段5 子表)。
type WorkOrderService interface {
	ListDispatchTickets(ctx context.Context) ([]DispatchTicket, error)
	// ListTicketItems 列表读模型:派单工单联表订单/客户/地址,供师傅端列表页。
	ListTicketItems(ctx context.Context) ([]TicketItem, error)
	// GetTicketItemByNo 详情读模型:同 ListTicketItems 联表语义,按 ticketNo 寻址单行。
	GetTicketItemByNo(ctx context.Context, ticketNo string) (*TicketItem, error)
	GetDispatchTicketByNo(ctx context.Context, ticketNo string) (*DispatchTicket, error)
	// AssignDispatchTicket 指派师傅(workerID/workerName 回填工单)。
	AssignDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error
	AssignPendingDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error
	// ClaimDispatchTicket 师傅领取:回填师傅并 PENDING→DOING。
	ClaimDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error
	CreateDispatchTicket(ctx context.Context, t DispatchTicket) (int64, error)
	ListComplaints(ctx context.Context) ([]Complaint, error)
	CreateComplaint(ctx context.Context, c Complaint) (int64, error)
	// CloseComplaint 投诉办结(ticketNo 寻址,status→CLOSED)。
	CloseComplaint(ctx context.Context, ticketNo string) error
	ListScanLogs(ctx context.Context, orderID int64) ([]ScanLog, error)
	AppendScanLog(ctx context.Context, l ScanLog) (int64, error)
}
