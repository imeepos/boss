package adminapi

// 阶段9 经营分析路由:五大指标/热力图/维护清单/自动报告(承接 admin analytics.html、report.html)。
// 全部 handler 实现见 analytics_handlers.go;此处只保留扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerAnalyticsRoutes 注册经营分析路由(menu:analytics)。
func registerAnalyticsRoutes(g *gin.RouterGroup, a *app.Application) {
	an := g.Group("", requirePerm(a.User, "menu:analytics"))

	an.GET("/analytics/indicators", analyticsFiveIndicatorsHandler(a))
	an.GET("/analytics/heatmap", analyticsHeatmapHandler(a))
	an.GET("/analytics/maintenance", analyticsMaintenanceHandler(a))
}

// registerReportRoutes 注册自动报告路由(menu:report)。
func registerReportRoutes(g *gin.RouterGroup, a *app.Application) {
	rp := g.Group("", requirePerm(a.User, "menu:report"))

	rp.GET("/db-patrol/orphans", reportPatrolOrphansHandler(a))
	rp.GET("/db-patrol/latest", reportLatestPatrolHandler(a))
	rp.GET("/reports", reportListHandler(a))
	rp.GET("/reports/latest", reportLatestHandler(a))
	rp.GET("/reports/recon/latest", reportReconLatestHandler(a))
	rp.GET("/reports/history", reportHistoryHandler(a))
	rp.GET("/compensation-tasks", compensationTasksHandler(a))
	rp.POST("/reports/:reportId/send", reportSendHandler(a))
}