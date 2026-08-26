package adminapi

// 配置下发域路由 handler 实现(承接 registerProvisionRoutes)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// provisionListTemplatesHandler GET /provision-templates:模板列表。
func provisionListTemplatesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Provision.ListTemplates(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// provisionCreateTemplateHandler POST /provision-templates:新建/复制配置模板(provision.yaml createProvisionTemplate)。
func provisionCreateTemplateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t provision.Template
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(t.LegalEntityID, "legalEntityId"),
				httpx.RequireString(t.Code, "code", 64),
				httpx.RequireString(t.Name, "name", 128),
				httpx.RequireEnumOrDefault(&t.Status, "status", "ENABLED", "ENABLED", "DISABLED"),
			)
		}) {
			return
		}
		id, err := a.Provision.CreateTemplate(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "provision_template", fmt.Sprint(id), map[string]any{"code": t.Code})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// provisionListTasksHandler GET /provision-tasks:任务列表。
func provisionListTasksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Provision.ListTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// provisionListLogsHandler GET /provision-logs:任务日志(按 taskId 过滤)。
func provisionListLogsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Provision.ListLogs(c.Request.Context(), queryInt64(c, "taskId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// provisionRetryTaskHandler POST /provision-tasks/{taskNo}/retry:失败任务重试(FAILED→PENDING + 重试计数留痕)。
func provisionRetryTaskHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskNo := c.Param("taskNo")
		if taskNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "taskNo is required"})
			return
		}
		task, err := a.Provision.GetTaskByNo(c.Request.Context(), taskNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		retries, err := latestRetries(a, c, task.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Provision.RetryTask(c.Request.Context(), task.ID, retries); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "provision.retry", "provision_task", task.TaskNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
