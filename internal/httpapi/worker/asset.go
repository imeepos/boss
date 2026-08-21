package workerapi

// W 师傅端门户资产域(worker/asset.yaml):拆机/换件/回收/领料/工具/维护/测速/资源。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerPortalAssetRoutes 资产域路由(wauth 组)。
func registerWorkerPortalAssetRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/tickets/:ticketNo/dismantle/scan", workerDismantleScanHandler(a))
	g.GET("/tickets/:ticketNo/replace", workerReplaceGetHandler)
	g.POST("/tickets/:ticketNo/replace", workerAuditOK(a, "replace"))
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

// workerReplaceGetHandler 换件预取:换件台账表缺失,返回占位步骤(缺口见报告)。
func workerReplaceGetHandler(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{
		"ticketNo": c.Param("ticketNo"), "oldEpc": "", "oldEpcStatus": "",
		"replaceType": "", "steps": []gin.H{{"name": "扫旧件", "status": "TODO"}},
	})
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
