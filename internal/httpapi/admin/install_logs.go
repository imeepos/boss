package adminapi

// 施工回单路由(admin 端只读 + worker 端上报)。
//
//   admin:
//     GET /install-logs?ticketId=123   列回单
//   worker:
//     POST /install-logs               师傅提交回单
//     POST /dispatch-tickets/:ticketNo/arrive  师傅到场打卡
//
// 决策依据:docs/notes/adopted/2026-08-28-procurement-install-gis-linkage.md §决策 2。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

func registerInstallLogRoutes(g *gin.RouterGroup, a *app.Application) {
	// 施工回单属履约管理面:menu:install-board(000165)门禁;师傅端回单走 worker 通道不经此处。
	g.GET("/install-logs", requirePerm(a.User, "menu:install-board"), installLogListHandler(a))
}
