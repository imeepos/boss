package provision

import (
	"context"
	"time"
)

// Template 下发模板(各公司设备型号不同,模板挂公司)。
type Template struct {
	ID            int64
	LegalEntityID int64
	Code          string // TPL-FTTH
	Name          string
}

// Task 下发任务(按模板向 LO 账号下发配置)。
type Task struct {
	ID          int64
	LoAccountID int64 // 软引用 lo_accounts
	TemplateID  int64
	Status      string // PENDING/DOING/DONE/FAILED
}

// Log 下发日志(下发任务对目标设备的结果/重试流水)。
type Log struct {
	ID           int64
	TaskID       int64
	ResourceID   int64 // 软引用 resources
	ResourceCode string
	TemplateID   int64
	TemplateCode string
	Result       string // SUCCESS/FAILED
	Retries      int16
	CreatedAt    time.Time
}

// ProvisionService 配置下发域服务口(阶段7)。
type ProvisionService interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	CreateTemplate(ctx context.Context, t Template) (int64, error)
	ListTasks(ctx context.Context) ([]Task, error)
	CreateTask(ctx context.Context, t Task) (int64, error)
	ListLogs(ctx context.Context, taskID int64) ([]Log, error)
	AppendLog(ctx context.Context, l Log) (int64, error)
}
