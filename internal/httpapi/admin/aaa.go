package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerAaaRoutes 注册认证计费域路由(账号、话单、认证日志和运行总览)。
func registerAaaRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/aaa/summary", requirePerm(a.User, "menu:aaadashboard"), aaaSummaryHandler(a))
	g.GET("/lo-accounts", requirePerm(a.User, "menu:loaccount"), aaaLoAccountsPageHandler(a))
	g.POST("/lo-accounts", requirePerm(a.User, "menu:loaccount"), loAccountCreateHandler(a)) // T2 建号(loid 唯一 40900,offer PUBLISHED)
	g.POST("/lo-accounts/:loid/reset-password", requirePerm(a.User, "menu:loaccount"), aaaResetPasswordHandler(a))
	g.GET("/cdrs", requirePerm(a.User, "menu:aaalog"), aaaCdrsPageHandler(a))
	g.GET("/auth-logs", requirePerm(a.User, "menu:aaalog"), aaaAuthLogsPageHandler(a))
	g.GET("/aaa/sessions", requirePerm(a.User, "menu:loaccount"), aaaSessionsPageHandler(a))
	g.POST("/aaa/sessions/:sessionId/disconnect", requirePerm(a.User, "menu:loaccount"), aaaSessionDisconnectHandler(a))
	// AAA-A5:NAS 客户端注册表(per-NAS 密钥/厂商/CoA 端口/启停),权限码沿用 menu:loaccount。
	g.GET("/aaa/nas", requirePerm(a.User, "menu:loaccount"), aaaNasPageHandler(a))
	g.POST("/aaa/nas", requirePerm(a.User, "menu:loaccount"), aaaNasCreateHandler(a))
	g.GET("/aaa/nas/:id", requirePerm(a.User, "menu:loaccount"), aaaNasGetHandler(a))
	g.PUT("/aaa/nas/:id", requirePerm(a.User, "menu:loaccount"), aaaNasUpdateHandler(a))
	g.DELETE("/aaa/nas/:id", requirePerm(a.User, "menu:loaccount"), aaaNasDeleteHandler(a))
}
