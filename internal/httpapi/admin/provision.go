package adminapi

// 配置下发域路由注册(承接 provision.yaml)。
// 全部 handler 实现见 provision_handlers.go;此处只保留扁平路由表 + 重试计数辅助。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerProvisionRoutes 注册配置下发域路由(承接 provision.yaml)。
func registerProvisionRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/provision-templates", requirePerm(a.User, "menu:template"), provisionListTemplatesHandler(a))
	g.POST("/provision-templates", requirePerm(a.User, "menu:template"), provisionCreateTemplateHandler(a))
	g.GET("/provision-tasks", requirePerm(a.User, "menu:provision"), provisionListTasksHandler(a))
	g.GET("/provision-logs", requirePerm(a.User, "menu:provlog"), provisionListLogsHandler(a))
	g.POST("/provision-tasks/:taskNo/retry", requirePerm(a.User, "menu:provision"), provisionRetryTaskHandler(a))
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
