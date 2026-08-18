package provision

import (
	"context"
	"time"
)

// Template 下发模板(各公司设备型号不同,模板挂公司)。
type Template struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Code          string `json:"code"` // TPL-FTTH
	Name          string `json:"name"`
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
	CreatedAt    time.Time `json:"createdAt"`
}

// ProvisionService 配置下发域服务口(阶段7)。
type ProvisionService interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	CreateTemplate(ctx context.Context, t Template) (int64, error)
	ListTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, t Task) (int64, error)
	// GetTaskByNo 按外部 task_no 寻址(契约 provision/v1 GetTask/RetryTask)。
	GetTaskByNo(ctx context.Context, taskNo string) (*Task, error)
	ListLogs(ctx context.Context, taskID int64) ([]Log, error)
	AppendLog(ctx context.Context, l Log) (int64, error)

	// ExecuteTask 执行下发:PENDING→DOING→DONE + SUCCESS 留痕(设备协议交互由 provisioner 执行)。
	ExecuteTask(ctx context.Context, taskID int64) error
	// FailTask 失败:→FAILED + 原因留痕。
	FailTask(ctx context.Context, taskID int64, reason string) error
	// RetryTask 失败重试:FAILED→PENDING + 重试计数留痕。
	RetryTask(ctx context.Context, taskID int64, retries int16) error
}
