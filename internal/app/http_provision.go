package app

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerProvisionRoutes 注册配置下发域路由(承接 provision.yaml)。
func registerProvisionRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/provision-templates", requirePerm(a.User, "menu:template"), func(c *gin.Context) {
		list, err := a.Provision.ListTemplates(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
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
}
