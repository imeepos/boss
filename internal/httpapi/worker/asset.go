package workerapi

// W 师傅端门户资产域(worker/asset.yaml):拆机/换件/回收/领料/工具/维护/测速/资源。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
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
	g.GET("/tickets/:ticketNo/measure", workerMeasureHandler)
	g.GET("/tickets/:ticketNo/resources", workerResourcesHandler(a))
}

// workerEPCReq 扫码请求体(拆机/换件共用)。
type workerEPCReq struct {
	EPC string `json:"epc" binding:"required"`
}

// workerDismantleScanHandler 拆机扫码解绑:复用 quadlink 强制扫码约束。
func workerDismantleScanHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req workerEPCReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
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
		if err := c.ShouldBindJSON(&req); err != nil || req.NewEpc == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		tk, _, err := ticketOrder(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		workerID, _ := portalWorker(c)
		if _, err := a.WorkerEvent.AppendReplaceLog(c.Request.Context(), worker.ReplaceLog{
			WorkerID: workerID, TicketNo: tk.TicketNo, OldEpc: req.OldEpc, NewEpc: req.NewEpc, CreatedAt: time.Now(),
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
			WorkerID: workerID, Name: item.Name + " " + item.Spec, Qty: 1,
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
		latest := make(map[string]bool, len(records))
		for _, r := range records { // ListTools 按 id 升序,后写覆盖 = 最新状态
			latest[r.Name] = r.Borrowed
		}
		items := make([]gin.H, 0, len(catalog))
		for _, t := range catalog {
			items = append(items, gin.H{
				"toolId": strconv.FormatInt(t.ID, 10), "name": t.Name,
				"code": t.Code, "borrowed": latest[t.Name],
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
			WorkerID: workerID, Name: tool.Name, Borrowed: borrowed,
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

// workerMeasureHandler 现场测速:实测通道未接,返回占位(缺口见报告)。
func workerMeasureHandler(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{
		"opticalPowerDbm": 0, "opticalPowerLabel": "", "downloadMbps": 0,
		"uploadMbps": 0, "packetLossRate": 0,
	})
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
