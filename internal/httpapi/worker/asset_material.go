package workerapi

// 物料领用与工具借还(从 asset.go 拆出,保持单文件 ≤300 行)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/pkg/apitypes"
)

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

// workerToolsHandler 工具借还状态:主档目录 × 师傅最新借还状态(优先 tool_id,历史行按名兜底)。
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
