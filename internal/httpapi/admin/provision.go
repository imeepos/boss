package adminapi

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerProvisionRoutes 注册配置下发域路由(承接 provision.yaml)。
func registerProvisionRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/provision-templates", requirePerm(a.User, "menu:template"), func(c *gin.Context) {
		list, err := a.Provision.ListTemplates(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 新建/复制配置模板(provision.yaml createProvisionTemplate,原 planned)。
	g.POST("/provision-templates", requirePerm(a.User, "menu:template"), func(c *gin.Context) {
		var t provision.Template
		if !httpx.BindAndValidate(c, &t, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(t.LegalEntityID, "legalEntityId"),
				httpx.RequireString(t.Code, "code", 64),
				httpx.RequireString(t.Name, "name", 128),
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
	})

	g.GET("/provision-tasks", requirePerm(a.User, "menu:provision"), func(c *gin.Context) {
		list, err := a.Provision.ListTasks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/provision-logs", requirePerm(a.User, "menu:provlog"), func(c *gin.Context) {
		list, err := a.Provision.ListLogs(c.Request.Context(), queryInt64(c, "taskId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 失败任务重试:按 taskNo 寻址,FAILED→PENDING + 重试计数留痕。
	g.POST("/provision-tasks/:taskNo/retry", requirePerm(a.User, "menu:provision"), func(c *gin.Context) {
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
	})
}

// latestRetries 取任务日志中的最大重试计数,供 RetryTask 递增留痕。
func latestRetries(a *app.Application, c *gin.Context, taskID int64) (int16, error) {
	logs, err := a.Provision.ListLogs(c.Request.Context(), taskID)
	if err != nil {
		return 0, err
	}
	var max int16
	for _, l := range logs {
		if l.Retries > max {
			max = l.Retries
		}
	}
	return max, nil
}
