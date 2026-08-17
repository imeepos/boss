package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerDeviceRoutes 注册设备监控/告警域路由(承接 alarm.yaml + device.html)。
func registerDeviceRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/alarms", requirePerm(a.User, "menu:alarm"), func(c *gin.Context) {
		list, err := a.Alarm.ListAlarms(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.POST("/alarms/:alarmId/ack", requirePerm(a.User, "menu:alarm"), func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("alarmId"), 10, 64)
		if err := a.Alarm.UpdateAlarmStatus(c.Request.Context(), id, "ACKED"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	g.GET("/device/metrics", requirePerm(a.User, "menu:device"), func(c *gin.Context) {
		list, err := a.Device.ListMetrics(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/device/maintenances", requirePerm(a.User, "menu:device"), func(c *gin.Context) {
		list, err := a.Device.ListMaintenances(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
