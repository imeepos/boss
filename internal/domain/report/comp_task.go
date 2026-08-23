package report

// S2 统一补偿任务/责任队列:失败来源汇聚 + 人工/自动补偿工作流。
// 实体对应 compensation_tasks 表,提供状态迁移、审计、优先级、SLA 管理等。

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// 优先级枚举。
const (
	PriorityLow    = "LOW"
	PriorityNormal = "NORMAL"
	PriorityHigh   = "HIGH"
	PriorityUrgent = "URGENT"
)

// 状态枚举。
const (
	TaskStatusOpen    = "OPEN"
	TaskStatusClaimed = "CLAIMED"
	TaskStatusDoing   = "DOING"
	TaskStatusDone    = "DONE"
	TaskStatusClosed  = "CLOSED"
)

// CompTask 统一补偿任务实体。
type CompTask struct {
	ID            int64        `json:"id"`
	Source        string       `json:"source"`  // 失败来源
	BizType       string       `json:"bizType"` // 业务对象类型
	BizID         string       `json:"bizId"`   // 业务对象ID
	FailureReason string       `json:"failureReason"`
	Priority      string       `json:"priority"` // LOW/NORMAL/HIGH/URGENT
	Status        string       `json:"status"`   // OPEN/CLAIMED/DOING/DONE/CLOSED
	AssigneeID    *int64       `json:"assigneeId,omitempty"`
	AssigneeName  string       `json:"assigneeName,omitempty"`
	SLADeadline   *time.Time   `json:"slaDeadline,omitempty"`
	ClaimedAt     *time.Time   `json:"claimedAt,omitempty"`
	ClaimedBy     *int64       `json:"claimedBy,omitempty"`
	ClosedAt      *time.Time   `json:"closedAt,omitempty"`
	ClosedBy      *int64       `json:"closedBy,omitempty"`
	CloseReason   string       `json:"closeReason,omitempty"`
	RetryCount    int          `json:"retryCount"`
	MaxRetries    int          `json:"maxRetries"`
	LastRetryAt   *time.Time   `json:"lastRetryAt,omitempty"`
	AuditLog      []AuditEntry `json:"auditLog"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

// AuditEntry 审计事件条目。
type AuditEntry struct {
	At     time.Time `json:"at"`
	Actor  string    `json:"actor"`  // 操作人account_name
	Action string    `json:"action"` // claim/transfer/retry/replay/close/create
	Detail string    `json:"detail"`
}

// ValidPriorities 合法优先级。
var ValidPriorities = []string{PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent}

// ValidStatuses 合法状态。
var ValidStatuses = []string{TaskStatusOpen, TaskStatusClaimed, TaskStatusDoing, TaskStatusDone, TaskStatusClosed}

// ErrIllegalStatus 非法状态迁移。
var ErrIllegalStatus = errors.New("comp: illegal status transition")

// allowedTransitions 允许的状态迁移。
var allowedTransitions = map[string][]string{
	TaskStatusOpen:    {TaskStatusClaimed, TaskStatusClosed},
	TaskStatusClaimed: {TaskStatusDoing, TaskStatusClosed, TaskStatusOpen},
	TaskStatusDoing:   {TaskStatusDone, TaskStatusClosed, TaskStatusOpen},
	TaskStatusDone:    {TaskStatusClosed},
	TaskStatusClosed:  {},
}

// CanTransition 检查状态迁移是否合法。
func CanTransition(from, to string) bool {
	tos, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, t := range tos {
		if t == to {
			return true
		}
	}
	return false
}

// AppendAudit 追加审计事件。
func AppendAudit(log []AuditEntry, actor, action, detail string) []AuditEntry {
	entry := AuditEntry{At: time.Now(), Actor: actor, Action: action, Detail: detail}
	return append(log, entry)
}

// SanitizeAudit 确保审计日志为合法JSON数组(抗拒空指针/非法类型)。
func SanitizeAudit(raw []byte) []AuditEntry {
	if len(raw) == 0 {
		return []AuditEntry{}
	}
	var entries []AuditEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return []AuditEntry{}
	}
	if entries == nil {
		return []AuditEntry{}
	}
	return entries
}

// CompTaskSource 来源常量。
const (
	SourceOrder     = "order"
	SourcePayment   = "payment"
	SourceCDR       = "cdr"
	SourceTax       = "tax"
	SourceQuadlink  = "quadlink"
	SourceProvision = "provision"
	SourcePoints    = "points"
	SourceCoupon    = "coupon"
	SourceWebhook   = "webhook"
	SourcePatrol    = "patrol"
)

// CompTaskStore 补偿任务存储接口(窄口,不与 Store 合并)。
type CompTaskStore interface {
	ListCompTasks(ctx context.Context, filter CompTaskFilter) ([]CompTask, int, error)
	GetCompTask(ctx context.Context, id int64) (*CompTask, error)
	CreateCompTask(ctx context.Context, t *CompTask) error
	UpdateCompTask(ctx context.Context, t *CompTask) error
	ClaimCompTask(ctx context.Context, id, actorID int64, actorName string) error
	TransferCompTask(ctx context.Context, id, fromID, toID int64, toName, actorName string) error
	CloseCompTask(ctx context.Context, id, actorID int64, actorName, reason string) error
	IncrementRetry(ctx context.Context, id int64) error
	BatchInsertCompTasks(ctx context.Context, tasks []CompTask) error
}

// CompTaskFilter 补偿任务查询过滤。
type CompTaskFilter struct {
	Status     string
	Source     string
	Priority   string
	AssigneeID *int64
	BizType    string
	Offset     int
	Limit      int
}

// DefaultCompLimit 默认分页大小。
const DefaultCompLimit = 20

// ErrCompTaskNotFound 任务不存在。
var ErrCompTaskNotFound = errors.New("comp: task not found")

// ErrCompTaskStoreNotSupported 存储不支持补偿任务(如测试 fake)。
var ErrCompTaskStoreNotSupported = errors.New("comp: store does not support compensation tasks")

// CompTaskService 补偿任务服务(ReportService 的可选能力)。
type CompTaskService struct {
	St CompTaskStore
}

// NewCompTaskService 构造。
func NewCompTaskService(st CompTaskStore) *CompTaskService {
	return &CompTaskService{St: st}
}

// List 查询过滤。
func (s *CompTaskService) List(ctx context.Context, f CompTaskFilter) ([]CompTask, int, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = DefaultCompLimit
	}
	return s.St.ListCompTasks(ctx, f)
}

// Get 按ID取。
func (s *CompTaskService) Get(ctx context.Context, id int64) (*CompTask, error) {
	return s.St.GetCompTask(ctx, id)
}

// Claim 领取任务。
func (s *CompTaskService) Claim(ctx context.Context, id, actorID int64, actorName string) error {
	return s.St.ClaimCompTask(ctx, id, actorID, actorName)
}

// Transfer 转派任务。
func (s *CompTaskService) Transfer(ctx context.Context, id, fromID, toID int64, toName, actorName string) error {
	return s.St.TransferCompTask(ctx, id, fromID, toID, toName, actorName)
}

// Close 关闭任务。
func (s *CompTaskService) Close(ctx context.Context, id, actorID int64, actorName, reason string) error {
	return s.St.CloseCompTask(ctx, id, actorID, actorName, reason)
}

// Retry 重试(计数+1)。
func (s *CompTaskService) Retry(ctx context.Context, id int64) error {
	return s.St.IncrementRetry(ctx, id)
}

// Create 创建任务。
func (s *CompTaskService) Create(ctx context.Context, t *CompTask) error {
	return s.St.CreateCompTask(ctx, t)
}

// BatchCreate 批量创建。
func (s *CompTaskService) BatchCreate(ctx context.Context, tasks []CompTask) error {
	if len(tasks) == 0 {
		return nil
	}
	return s.St.BatchInsertCompTasks(ctx, tasks)
}
