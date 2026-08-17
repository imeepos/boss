package app

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerAaaRoutes 注册认证计费域路由(承接 oss.yaml listLoAccounts + aaa.yaml listAaaLogs)。
func registerAaaRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/lo-accounts", requirePerm(a.User, "menu:loaccount"), func(c *gin.Context) {
		list, err := a.Aaa.ListLoAccounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/cdrs", requirePerm(a.User, "menu:aaalog"), func(c *gin.Context) {
		list, err := a.Aaa.ListCdrs(c.Request.Context(), c.Query("loid"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/auth-logs", requirePerm(a.User, "menu:aaalog"), func(c *gin.Context) {
		list, err := a.Aaa.ListAuthLogs(c.Request.Context(), c.Query("loid"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
