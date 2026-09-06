package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerAaaRoutes 注册认证计费域路由(账号、话单、认证日志和运行总览)。
func registerAaaRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/aaa/summary", requirePerm(a.User, "menu:aaadashboard"), aaaSummaryHandler(a))
	g.GET("/lo-accounts", requirePerm(a.User, "menu:loaccount"), aaaLoAccountsPageHandler(a))
	g.POST("/lo-accounts/:loid/reset-password", requirePerm(a.User, "menu:loaccount"), aaaResetPasswordHandler(a))
	g.GET("/cdrs", requirePerm(a.User, "menu:aaalog"), aaaCdrsPageHandler(a))
	g.GET("/auth-logs", requirePerm(a.User, "menu:aaalog"), aaaAuthLogsPageHandler(a))
}
