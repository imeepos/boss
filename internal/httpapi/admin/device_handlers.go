// device 域具名 handler(承接 registerDeviceRoutes 扁平路由表)。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// deviceListAlarmsHandler GET /alarms:告警列表(可选 resourceId 过滤)。
func deviceListAlarmsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Alarm.ListAlarms(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// deviceAckAlarmHandler POST /alarms/{alarmId}/ack:告警 ACKED。
func deviceAckAlarmHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "alarmId")
		if !ok {
			return
		}
		if err := a.Alarm.UpdateAlarmStatus(c.Request.Context(), id, "ACKED"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// deviceBatchRetestHandler POST /alarms/batch-retest:台风应急·批量复测。
func deviceBatchRetestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req batchRetestReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		taskNo, err := a.Alarm.AppendRetestTask(c.Request.Context(), req.Scope)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "alarm.batch-retest", "alarm_retest", taskNo, map[string]any{"scope": req.Scope})
		respond(c, apitypes.CodeOK, gin.H{"taskNo": taskNo, "scope": req.Scope})
	}
}

// deviceListMetricsHandler GET /device/metrics:设备指标列表(可选 resourceId)。
func deviceListMetricsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Device.ListMetrics(c.Request.Context(), queryInt64(c, "resourceId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// deviceListMaintenancesHandler GET /device/maintenances:设备维护记录列表。
func deviceListMaintenancesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Device.ListMaintenances(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
