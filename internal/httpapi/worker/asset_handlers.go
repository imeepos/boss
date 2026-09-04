package workerapi

// W 师傅端资产域 handler 实现(asset.go 仅留路由表 + 共享类型/视图工具)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// workerDismantleScanHandler 拆机扫码解绑:复用 quadlink 强制扫码约束。
func workerDismantleScanHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerEPCReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		if err := a.QuadLink.UnbindRequireScan(c.Request.Context(), tk.OrderID, req.EPC); err != nil {
			httpx.RespondScanErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"unbound": true, "portReleased": true})
	}
}

// workerReplaceGetHandler 换件预取:旧件 EPC 经四码绑定链(地址→quadlink→资产→标签)取真实值,
// 步骤状态由换件流水(worker_replace_logs)驱动。
func workerReplaceGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		oldEpc, oldStatus := boundEpcOf(a, c, ord.AddressID)
		logs, err := a.WorkerEvent.ListReplaceLogs(c.Request.Context(), tk.TicketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		done := len(logs) > 0
		respond(c, apitypes.CodeOK, gin.H{
			"ticketNo": tk.TicketNo, "oldEpc": oldEpc, "oldEpcStatus": oldStatus,
			"replaceType": "故障调换 · 旧件返修",
			"steps": []gin.H{
				{"name": "扫旧件", "status": done, "note": oldEpc},
				{"name": "换新件", "status": done, "note": lastNewEpc(logs)},
				{"name": "登记返修", "status": done, "note": ""},
			},
		})
	}
}

// workerReplacePostHandler 完成换件:落 worker_replace_logs + 审计留痕。
func workerReplacePostHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerReplaceReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.NewEpc, "newEpc", 64),
			)
		}) {
			return
		}
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := workerAppendReplaceLog(c, a, tk, req); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "worker_replace", tk.TicketNo,
			map[string]any{"oldEpc": req.OldEpc, "newEpc": req.NewEpc})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerAssetReturnHandler 旧件返库登记:落 asset_returns(PENDING 待确认)。
func workerAssetReturnHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		snap, err := a.WorkerEvent.ResolveFactSnapshot(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		_, err = a.WorkerEvent.AppendAssetReturn(c.Request.Context(), workerAssetReturnOf(c, snap))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerMaintenanceHandler 维护清单(device_maintenances,按优先级排序展示)。
func workerMaintenanceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Device.ListMaintenances(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": workerMaintenanceItems(list)})
	}
}

// workerMaintenanceItems 维护项视图(ageYears 取指针值)。
func workerMaintenanceItems(list []device.DeviceMaintenance) []gin.H {
	items := make([]gin.H, 0, len(list))
	for _, m := range list {
		age := 0
		if m.AgeYears != nil {
			age = int(*m.AgeYears)
		}
		items = append(items, gin.H{
			"deviceNo": m.DeviceNo, "deviceType": m.DeviceType,
			"healthScore": m.HealthScore, "faultCount": m.FaultCount,
			"ageYears": age, "reason": m.Reason, "priority": m.Priority,
			"priorityLabel": portalPriorityLabel(m.Priority),
		})
	}
	return items
}

// workerMeasureHandler 现场测速:光功率/丢包取设备采集链(device_metrics,
// 地址→资源→最新样本),上下行速率暂无实测通道(保持 0,如实回传)。
func workerMeasureHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		optical, loss, label := latestDeviceSample(a, c, ord.AddressID)
		respond(c, apitypes.CodeOK, gin.H{
			"opticalPowerDbm": optical, "opticalPowerLabel": label,
			"downloadMbps": 0, "uploadMbps": 0, "packetLossRate": loss,
		})
	}
}

// latestDeviceSample 取地址所属资源的最新的设备样本;无样本返回 0 值 + 离线标签。
func latestDeviceSample(a *app.Application, c *gin.Context, addressID int64) (float64, float64, string) {
	latest := findLatestDeviceMetric(a, c, addressID)
	if latest == nil {
		return 0, 0, "无数据"
	}
	optical, loss := sampleReading(latest)
	return optical, loss, opticalPowerLabel(optical)
}

// findLatestDeviceMetric 跨资源遍历取最新样本;无样本返回 nil。
func findLatestDeviceMetric(a *app.Application, c *gin.Context, addressID int64) *device.DeviceMetric {
	resources, err := a.Resource.ListResources(c.Request.Context())
	if err != nil {
		return nil
	}
	var latest *device.DeviceMetric
	for _, r := range resources {
		if r.AddressID != addressID {
			continue
		}
		metrics, err := a.Device.ListMetrics(c.Request.Context(), r.ID)
		if err != nil {
			continue
		}
		for i := range metrics {
			m := metrics[i]
			if latest == nil || m.CollectedAt.After(latest.CollectedAt) {
				latest = &m
			}
		}
	}
	return latest
}

// sampleReading 设备样本光功率 + 丢包率。
func sampleReading(m *device.DeviceMetric) (float64, float64) {
	optical, loss := 0.0, 0.0
	if m.OpticalPower != nil {
		optical = *m.OpticalPower
	}
	if m.PacketLoss != nil {
		loss = *m.PacketLoss
	}
	return optical, loss
}

// opticalPowerLabel 光功率判档:-24dBm 以上正常,-27 以下异常,其间偏弱。
func opticalPowerLabel(dbm float64) string {
	switch {
	case dbm == 0:
		return "无数据"
	case dbm >= -24:
		return "正常"
	case dbm >= -27:
		return "偏弱"
	default:
		return "异常"
	}
}

// workerResourcesHandler 片区资源:经 resource 域核查目标地址空闲端口。
func workerResourcesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, ord, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		_, options, err := a.Resource.Check(c.Request.Context(), ord.AddressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"idlePorts": len(options), "nearestSplitter": "", "idlePonPorts": options,
			"ticketNo": tk.TicketNo,
		})
	}
}
