package adminapi

// 施工看板与回单路由(admin 端只读 + worker 端上报)。
//
//   admin:
//     GET /dispatch-tickets           施工看板·全量派单工单(含到场打卡事实)
//     GET /install-logs?ticketId=123  列回单
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
	// 施工看板列表归 menu:install-board(000165)门禁,刻意不挂 /dispatch 组的
	// menu:dispatch:technician 持有后者但按 000168 契约不持看板读权,挂错门禁
	// 会重开「technician 读全量工单+打卡事实」的越权面。
	g.GET("/dispatch-tickets", requirePerm(a.User, "menu:install-board"), installBoardTicketsHandler(a))
	// 施工回单属履约管理面:menu:install-board(000165)门禁;师傅端回单走 worker 通道不经此处。
	g.GET("/install-logs", requirePerm(a.User, "menu:install-board"), installLogListHandler(a))
}
