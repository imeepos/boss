package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// batchRetestReq 批量复测请求体(scope=下发范围,必填)。
type batchRetestReq struct {
	Scope string `json:"scope" binding:"required"`
}

// registerDeviceRoutes 注册设备监控/告警域路由(承接 alarm.yaml + device.html)。
func registerDeviceRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/alarms", requirePerm(a.User, "menu:alarm"), deviceListAlarmsHandler(a))
	g.POST("/alarms/:alarmId/ack", requirePerm(a.User, "menu:alarm"), deviceAckAlarmHandler(a))
	// 台风应急·批量复测:按片区受理,返回批量复测任务号(alarm.yaml batchRetestAlarms)。
	g.POST("/alarms/batch-retest", requirePerm(a.User, "menu:alarm"), deviceBatchRetestHandler(a))
	g.GET("/device/metrics", requirePerm(a.User, "menu:device"), deviceListMetricsHandler(a))
	g.GET("/device/maintenances", requirePerm(a.User, "menu:device"), deviceListMaintenancesHandler(a))
}
