package provision

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Template 下发模板(各公司设备型号不同,模板挂公司)。
type Template struct {
	ID            int64          `json:"id"`
	LegalEntityID int64          `json:"legalEntityId"`
	Code          string         `json:"code"` // TPL-FTTH
	Name          string         `json:"name"`
	Content       map[string]any `json:"content"`
	Version       int32          `json:"version"`
	Status        string         `json:"status"` // ENABLED/DISABLED
	UpdatedAt     time.Time      `json:"updatedAt"`
	BoundOffers   int64          `json:"boundOffers"` // 已绑定此模板的套餐数(admin 可见性)
}

// OfferTemplateBinding 产品/套餐 → 下发模板绑定行(方案B)。
// 后台显式绑定,环节7 优先走绑定;无绑定才按带宽兜底。
type OfferTemplateBinding struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	OfferID       int64  `json:"offerId"`
	TemplateID    int64  `json:"templateId"`
	TemplateCode  string `json:"templateCode"`
	TemplateName  string `json:"templateName"`
	Remark        string `json:"remark"`
}

// Task 下发任务(按模板向 LO 账号下发配置)。
// TaskNo/OrderID/StageEvent 对齐契约 provision/v1(债务偿还:任务外部寻址 + 来源订单/环节事件)。
type Task struct {
	ID          int64  `json:"id"`
	TaskNo      string `json:"taskNo"`      // 外部稳定标识(EnqueueTask/GetTask/RetryTask 寻址)
	OrderID     int64  `json:"orderId"`     // 来源订单(0=手工任务)
	StageEvent  string `json:"stageEvent"`  // preConfigOLT/activateUser/notifyActivation
	LoAccountID int64  `json:"loAccountId"` // 软引用 lo_accounts
	TemplateID  int64  `json:"templateId"`
	Status      string `json:"status"` // PENDING/DOING/DONE/FAILED
}

// Log 下发日志(下发任务对目标设备的结果/重试流水)。
type Log struct {
	ID           int64     `json:"id"`
	TaskID       int64     `json:"taskId"`
	ResourceID   int64     `json:"resourceId"` // 软引用 resources
	ResourceCode string    `json:"resourceCode"`
	TemplateID   int64     `json:"templateId"`
	TemplateCode string    `json:"templateCode"`
	Result       string    `json:"result"` // SUCCESS/FAILED
	Retries      int16     `json:"retries"`
	Commands     string    `json:"commands"`       // 顺序执行的设备指令(换行分隔;空=无设备交互)
	DeviceResp   string    `json:"deviceResponse"` // 设备原始应答/错误
	CreatedAt    time.Time `json:"createdAt"`
}

// ExecTrace 一次设备交互的完整留痕:顺序指令 + 原始应答(执行器回传,落 provision_logs)。
type ExecTrace struct {
	Commands []string `json:"commands"`
	Response string   `json:"response"`
}

// JoinCommands 指令序列拼为落库文本(空序列→空串)。
func JoinCommands(cmds []string) string { return strings.Join(cmds, "\n") }

// LogOrderInfo 详情页订单维度(日志→任务→订单软引用,订单已删则为零值)。
type LogOrderInfo struct {
	OrderNo   string `json:"orderNo"`
	Status    string `json:"status"`
	OfferName string `json:"offerName"`
}

// LogTemplateInfo 详情页模板维度(模板已删则为零值)。
type LogTemplateInfo struct {
	Code    string         `json:"code"`
	Name    string         `json:"name"`
	Status  string         `json:"status"`
	Version int32          `json:"version"`
	Content map[string]any `json:"content"`
}

// LogDetail 日志详情页聚合视图:日志本体 + 任务/订单/模板三维上下文。
type LogDetail struct {
	Log      Log             `json:"log"`
	Task     Task            `json:"task"`
	Order    LogOrderInfo    `json:"order"`
	Template LogTemplateInfo `json:"template"`
}

// ErrLogNotFound 下发日志不存在。
var ErrLogNotFound = errors.New("provision: log not found")

// ProvisionService 配置下发域服务口(阶段7)。
type ProvisionService interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	CreateTemplate(ctx context.Context, t Template) (int64, error)
	UpdateTemplate(ctx context.Context, t Template) error
	SetTemplateStatus(ctx context.Context, id int64, status string) error
	DeleteTemplate(ctx context.Context, id int64) error
	ListTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, t Task) (int64, error)
	// GetTaskByNo 按外部 task_no 寻址(契约 provision/v1 GetTask/RetryTask)。
	GetTaskByNo(ctx context.Context, taskNo string) (*Task, error)
	ListLogs(ctx context.Context, taskID int64) ([]Log, error)
	AppendLog(ctx context.Context, l Log) (int64, error)
	// GetLogDetail 日志详情聚合(日志+任务+订单+模板);日志不存在返回 ErrLogNotFound。
	GetLogDetail(ctx context.Context, logID int64) (*LogDetail, error)

	// ClaimTask 原子领取一个 PENDING 任务(状态置 DOING)并返回;无待办返回 (nil, nil)。
	// 用 FOR UPDATE SKIP LOCKED 防止多 provisioner 实例重复下发同一任务。
	ClaimTask(ctx context.Context) (*Task, error)
	// ExecuteTask 完成下发:PENDING 或 DOING→DONE + SUCCESS 留痕(trace 为设备交互留痕)。
	ExecuteTask(ctx context.Context, taskID int64, trace ExecTrace) error
	// FailTask 失败:→FAILED + 原因与设备交互留痕。
	FailTask(ctx context.Context, taskID int64, reason string, trace ExecTrace) error
	// RetryTask 失败重试:FAILED→PENDING + 重试计数留痕。
	RetryTask(ctx context.Context, taskID int64, retries int16) error

	// 产品/套餐 ↔ 下发模板显式绑定(方案B)。
	// GetOfferBinding 查询套餐已绑模板;未绑定返回 (nil, nil)。
	GetOfferBinding(ctx context.Context, offerID int64) (*OfferTemplateBinding, error)
	// UpsertOfferBinding 绑定/改绑(幂等,同套餐 ON CONFLICT 更新);校验套餐/模板存在、同法人、模板启用。
	UpsertOfferBinding(ctx context.Context, offerID, templateID int64, remark string) (int64, error)
	// DeleteOfferBinding 解绑。
	DeleteOfferBinding(ctx context.Context, offerID int64) error
	// ListOfferBindings 列出全部绑定(admin 产品页列绑定状态用)。
	ListOfferBindings(ctx context.Context) ([]OfferTemplateBinding, error)
}
