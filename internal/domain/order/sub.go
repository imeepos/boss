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
	CreatedAt       string `json:"createdAt"`
	RemoteDiagnosis string `json:"remoteDiagnosis"`
	SlaDeadline     string `json:"slaDeadline"` // SLA 截止时间(派单时写入,空=无 SLA 或已过期)
}

// complaintTypeLabels complaints.type → 故障类型中文标签(对齐 docs/contract/complaint-type-map.md)。
var complaintTypeLabels = map[string]string{
	"SINGLE_OUTAGE": "单户断网（紧急 SLA ≤4h）",
	"PARTIAL_OUTAGE": "部分业务不可用（常规 SLA ≤8h）",
	"SLOW_NET":       "网速慢（一般 SLA ≤24h）",
	"WIFI_ISSUE":     "Wi-Fi 信号问题（一般 SLA ≤24h）",
	"DEVICE_FAULT":   "设备故障（紧急 SLA ≤4h）",
	"OTHER":          "其他问题（一般 SLA ≤24h）",
}

// FaultTypeLabel complaints.type → 中文标签,未知值 fallback。
func (c *Complaint) FaultTypeLabel() string {
	if lbl, ok := complaintTypeLabels[c.Type]; ok {
		return lbl
	}
	return "报障（待分类）"
}

// SlaHours complaints.type → SLA 小时数(用于未写入 sla_deadline 时实时兜底计算)。
var complaintSlaHours = map[string]int{
	"SINGLE_OUTAGE": 4, "PARTIAL_OUTAGE": 8, "SLOW_NET": 24,
	"WIFI_ISSUE": 24, "DEVICE_FAULT": 4, "OTHER": 24,
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

// TicketItem 派单工单详情读模型:联表 orders/customers/product_offers/addresses
// /user_addresses/order_stages + LEFT JOIN complaints(报障工单)。
//
// api/openapi/worker/schemas.yaml::TicketDetail 字段全部覆盖。
// 报障字段(faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis)仅当
// complaints 有对应记录时才有值;新装工单无 complaints 行则为空串/0。
type TicketItem struct {
	TicketID      int64  `json:"ticketId"`
	TicketNo      string `json:"ticketNo"`
	OrderID       int64  `json:"orderId"`
	WorkerID      int64  `json:"workerId"`
	Status        string `json:"status"`      // PENDING/DOING/DONE/CANCELED
	OrderStatus   string `json:"orderStatus"` // 订单终态兜底:DONE/CANCELED 时工单视图按完成处理
	CustomerName  string `json:"customerName"`
	CustomerPhone string `json:"customerPhone"` // 明文,handler 出门必须 httpx.MaskPhone 脱敏
	OfferName     string `json:"offerName"`     // → product_offers.name(下单快照)
	Address       string `json:"address"`
	Stage         int8   `json:"stage"` // 订单当前环节 1~12
	FinishedAt    string `json:"finishedAt"` // 订单 DONE 时 MAX(order_stages.finished_at) 派生,空=进行中

	// 装维工单字段(环节3/5/8 写入 dispatch_tickets)。
	SplitterPort string `json:"splitterPort"` // 分光器端口(环节3/5 端口预占时系统填入)
	PreBindTag   string `json:"preBindTag"`   // 预绑定 EPC 标签(环节5 标签预绑定时系统填入)
	ScheduleSlot string `json:"scheduleSlot"` // 预约时间段(环节8 派单时调度填入)

	// 报障工单字段(LEFT JOIN complaints,仅报障单有值)。
	ComplaintType   string `json:"complaintType"`   // complaints.type 原始值
	FaultTypeLabel  string `json:"faultTypeLabel"`   // complaintTypeLabels 映射后的中文标签
	ReportedAt      string `json:"reportedAt"`       // complaints.created_at
	SlaLeftMinutes  int    `json:"slaLeftMinutes"`   // SLA 剩余分钟(实时计算或从 sla_deadline 派生)
	RemoteDiagnosis string `json:"remoteDiagnosis"`  // complaints.remote_diagnosis
	SlaDeadline     string `json:"slaDeadline"`       // complaints.sla_deadline 原始值
}

// AssignOpt 指派/改约时可选写入工单的附加字段。
// ScheduleSlot:预约时间段(如 "08-22 14:00-16:00");PreBindTag:预绑定 EPC 标签。
// 两者均为空串时不覆盖已有值(幂等)。
type AssignOpt struct {
	ScheduleSlot string // 预约时间段
	PreBindTag   string // 预绑定 EPC 标签
}

// WorkOrderService 订单工单/报障/扫码域服务口(阶段5 子表)。
type WorkOrderService interface {
	ListDispatchTickets(ctx context.Context) ([]DispatchTicket, error)
	// ListTicketItems 列表读模型:派单工单联表订单/客户/地址,供师傅端列表页。
	ListTicketItems(ctx context.Context) ([]TicketItem, error)
	// GetTicketItemByNo 详情读模型:同 ListTicketItems 联表语义,按 ticketNo 寻址单行。
	GetTicketItemByNo(ctx context.Context, ticketNo string) (*TicketItem, error)
	GetDispatchTicketByNo(ctx context.Context, ticketNo string) (*DispatchTicket, error)
	// AssignDispatchTicket 指派师傅(workerID/workerName 回填工单);opt 可选写入预约/预绑定。
	AssignDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string, opt ...AssignOpt) error
	AssignPendingDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string, opt ...AssignOpt) error
	// ClaimDispatchTicket 师傅领取:回填师傅并 PENDING→DOING。
	ClaimDispatchTicket(ctx context.Context, ticketNo string, workerID int64, workerName string) error
	// UpdateScheduleSlot 改约:更新预约时间段 + 重置报障 SLA 截止时间。
	UpdateScheduleSlot(ctx context.Context, ticketNo string, scheduleSlot string) error
	CreateDispatchTicket(ctx context.Context, t DispatchTicket) (int64, error)
	ListComplaints(ctx context.Context) ([]Complaint, error)
	CreateComplaint(ctx context.Context, c Complaint) (int64, error)
	// CloseComplaint 投诉办结(ticketNo 寻址,status→CLOSED)。
	CloseComplaint(ctx context.Context, ticketNo string) error
	ListScanLogs(ctx context.Context, orderID int64) ([]ScanLog, error)
	AppendScanLog(ctx context.Context, l ScanLog) (int64, error)
}
