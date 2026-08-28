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
	g.GET("/install-logs", installLogListHandler(a))
}
