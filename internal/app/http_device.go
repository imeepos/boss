package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// batchRetestReq 批量复测请求体(scope=下发范围,必填)。
type batchRetestReq struct {
	Scope string `json:"scope" binding:"required"`
}

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

	// 台风应急·批量复测:按片区受理,返回批量复测任务号(alarm.yaml batchRetestAlarms)。
	g.POST("/alarms/batch-retest", requirePerm(a.User, "menu:alarm"), func(c *gin.Context) {
		var req batchRetestReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		taskNo, err := a.Alarm.AppendRetestTask(c.Request.Context(), req.Scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "alarm.batch-retest", "alarm_retest", taskNo, map[string]any{"scope": req.Scope})
		respond(c, apitypes.CodeOK, gin.H{"taskNo": taskNo, "scope": req.Scope})
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
