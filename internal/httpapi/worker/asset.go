package workerapi

// W 师傅端门户资产域(worker/asset.yaml):拆机/换件/回收/领料/工具/维护/测速/资源。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerPortalAssetRoutes 资产域路由(wauth 组)。
func registerWorkerPortalAssetRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/tickets/:ticketNo/dismantle/scan", workerDismantleScanHandler(a))
	g.GET("/tickets/:ticketNo/replace", workerReplaceGetHandler(a))
	g.POST("/tickets/:ticketNo/replace", workerReplacePostHandler(a))
	g.POST("/assets/:epc/return", workerAssetReturnHandler(a))
	g.GET("/materials", workerMaterialsHandler(a))
	g.POST("/materials/:itemId/out", workerMaterialOutHandler(a))
	g.GET("/materials/tools", workerToolsHandler(a))
	g.POST("/materials/tools/:toolId/borrow", workerToolBorrowHandler(a, true))
	g.POST("/materials/tools/:toolId/give-back", workerToolBorrowHandler(a, false))
	g.GET("/maintenance", workerMaintenanceHandler(a))
	g.GET("/tickets/:ticketNo/measure", workerMeasureHandler(a))
	g.GET("/tickets/:ticketNo/resources", workerResourcesHandler(a))
}

// workerEPCReq 扫码请求体(拆机/换件共用)。
type workerEPCReq struct {
	EPC string `json:"epc" binding:"required"`
}

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

// boundEpcOf 取地址当前绑定资产的标签 EPC;链路任一环缺失返回空串与 UNLINKED。
func boundEpcOf(a *app.Application, c *gin.Context, addressID int64) (string, string) {
	q, err := a.QuadLink.GetByAddress(c.Request.Context(), addressID)
	if err != nil || q == nil || q.AssetID == 0 {
		return "", "UNLINKED"
	}
	ast, err := a.Asset.GetAsset(c.Request.Context(), q.AssetID)
	if err != nil || ast == nil || ast.TagID == 0 {
		return "", q.Status
	}
	tags, err := a.Asset.ListTags(c.Request.Context())
	if err != nil {
		return "", q.Status
	}
	for _, t := range tags {
		if t.TagID == ast.TagID {
			return t.EpcCode, q.Status
		}
	}
	return "", q.Status
}

func lastNewEpc(logs []worker.ReplaceLog) string {
	if len(logs) == 0 {
		return ""
	}
	return logs[len(logs)-1].NewEpc
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
		workerID, _ := portalWorker(c)
		if _, err := a.WorkerEvent.AppendReplaceLog(c.Request.Context(), worker.ReplaceLog{
			WorkerID: workerID, DispatchTicketID: tk.TicketID, TicketNo: tk.TicketNo,
			OldEpc: req.OldEpc, NewEpc: req.NewEpc, CreatedAt: time.Now(),
		}); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "worker_replace", tk.TicketNo,
			map[string]any{"oldEpc": req.OldEpc, "newEpc": req.NewEpc})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerReplaceReq 换件提交请求体(契约 asset.yaml Replace)。
type workerReplaceReq struct {
	OldEpc string `json:"oldEpc"`
	NewEpc string `json:"newEpc"`
}

// workerAssetReturnHandler 旧件返库登记:落 asset_returns(PENDING 待确认)。
func workerAssetReturnHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		_, err := a.WorkerEvent.AppendAssetReturn(c.Request.Context(), workerAssetReturnOf(c, workerID))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerMaterialsHandler 今日领用清单 + 待返旧件。
func workerMaterialsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		mats, err := a.WorkerEvent.ListMaterials(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(mats))
		for _, m := range mats {
			items = append(items, gin.H{
				"itemId": strconv.FormatInt(m.ID, 10), "name": m.Name,
				"qty": m.Qty, "spec": "", "outBound": true,
			})
		}
		pending, _ := a.WorkerEvent.ListAssetReturns(c.Request.Context(), workerID)
		pendingItems := make([]gin.H, 0)
		for _, r := range pending {
			if r.Status == "PENDING" {
				pendingItems = append(pendingItems, gin.H{"epc": "", "reason": r.Reason})
			}
		}
		catalog := make([]gin.H, 0)
		if items, err := a.WorkerEvent.ListMaterialItems(c.Request.Context()); err == nil {
			for _, m := range items {
				catalog = append(catalog, gin.H{
					"itemId": strconv.FormatInt(m.ID, 10), "name": m.Name, "spec": m.Spec, "unit": m.Unit,
				})
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"items": items, "returned": gin.H{"repairCount": 0, "dismantleCount": 0},
			"pendingReturn": pendingItems, "catalog": catalog,
		})
	}
}

// workerMaterialOutHandler 领料出库:主档校验 + 真实名称落领用记录。
func workerMaterialOutHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		item := lookupMaterialItem(c, a)
		if item == nil {
			return
		}
		workerID, _ := portalWorker(c)
		_, err := a.WorkerEvent.AppendMaterial(c.Request.Context(), worker.Material{
			WorkerID: workerID, ItemID: item.ID, Name: item.Name + " " + item.Spec, Qty: 1,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "name": item.Name, "spec": item.Spec})
	}
}

// workerToolsHandler 工具借还状态:主档目录 × 师傅最新借还状态(同名取最后一条记录)。
func workerToolsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		catalog, err := a.WorkerEvent.ListToolItems(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		records, err := a.WorkerEvent.ListTools(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		latest := make(map[int64]bool, len(records))
		latestByName := make(map[string]bool, len(records))
		for _, r := range records { // ListTools 按 id 升序,后写覆盖 = 最新状态
			if r.ToolID > 0 {
				latest[r.ToolID] = r.Borrowed
			} else {
				latestByName[r.Name] = r.Borrowed // 历史行未回填 tool_id,按名兜底
			}
		}
		items := make([]gin.H, 0, len(catalog))
		for _, t := range catalog {
			borrowed, ok := latest[t.ID]
			if !ok {
				borrowed = latestByName[t.Name]
			}
			items = append(items, gin.H{
				"toolId": strconv.FormatInt(t.ID, 10), "name": t.Name,
				"code": t.Code, "borrowed": borrowed,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerToolBorrowHandler 工具借用/归还登记:主档校验 + 真实名称落记录。
func workerToolBorrowHandler(a *app.Application, borrowed bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tool := lookupToolItem(c, a)
		if tool == nil {
			return
		}
		workerID, _ := portalWorker(c)
		_, err := a.WorkerEvent.AppendTool(c.Request.Context(), worker.Tool{
			WorkerID: workerID, ToolID: tool.ID, Name: tool.Name, Borrowed: borrowed,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "name": tool.Name})
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
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalPriorityLabel 优先级中文(MUST_REPLACE/SUGGEST/WATCH)。
func portalPriorityLabel(p string) string {
	switch p {
	case "MUST_REPLACE":
		return "必须更换"
	case "SUGGEST":
		return "建议更换"
	default:
		return "观察"
	}
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
	resources, err := a.Resource.ListResources(c.Request.Context())
	if err != nil {
		return 0, 0, "无数据"
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
	if latest == nil {
		return 0, 0, "无数据"
	}
	optical, loss := 0.0, 0.0
	if latest.OpticalPower != nil {
		optical = *latest.OpticalPower
	}
	if latest.PacketLoss != nil {
		loss = *latest.PacketLoss
	}
	return optical, loss, opticalPowerLabel(optical)
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
