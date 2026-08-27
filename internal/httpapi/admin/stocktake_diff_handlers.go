package adminapi

// 盘点差异明细具名 handler(S10 补全:明细清单/扫码回填/逐条处置)。
// 路由注册见 asset.go;关单 handler(stocktakeHandleDiffHandler)在 asset_handlers.go。

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// assetScanStatuses 扫码可上报的资产状态(terms.md asset.status)。
var assetScanStatuses = []string{"IN_STOCK", "DEPLOYED", "MAINTENANCE", "SCRAPPED"}

// stocktakeItemActions 差异处置动作(S10:确认/修正/上报)。
var stocktakeItemActions = []string{"CONFIRM", "FIX", "ESCALATE"}

// stocktakeItemsHandler GET /stocktakes/{taskId}/items:盘点差异明细清单。
func stocktakeItemsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID, ok := httpx.ParsePathParamInt64(c, "taskId")
		if !ok {
			return
		}
		list, err := a.Asset.ListStocktakeItems(c.Request.Context(), taskID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

type stocktakeScanReq struct {
	AssetID int64  `json:"assetId"`
	Status  string `json:"status"` // 实盘所见状态
}

// stocktakeScanHandler POST /stocktakes/{taskId}/scans:回填一次扫码并重算进度/差异数。
func stocktakeScanHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID, ok := httpx.ParsePathParamInt64(c, "taskId")
		if !ok {
			return
		}
		var req stocktakeScanReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.AssetID, "assetId"),
				httpx.RequireEnum(req.Status, "status", assetScanStatuses...),
			)
		}) {
			return
		}
		itemID, kind, err := a.Asset.ScanStocktake(c.Request.Context(), taskID, req.AssetID, req.Status)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "stocktake_scan", fmt.Sprintf("%d/%d", taskID, req.AssetID),
			map[string]any{"status": req.Status, "kind": kind})
		respond(c, apitypes.CodeOK, gin.H{"itemId": itemID, "kind": kind})
	}
}

type stocktakeItemHandleReq struct {
	Action string `json:"action"` // CONFIRM/FIX/ESCALATE
	Note   string `json:"note"`   // FIX/ESCALATE 必填
}

// stocktakeItemHandleHandler POST /stocktakes/{taskId}/items/{itemId}/handle:逐条处置差异。
func stocktakeItemHandleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID, ok := httpx.ParsePathParamInt64(c, "taskId")
		if !ok {
			return
		}
		itemID, ok := httpx.ParsePathParamInt64(c, "itemId")
		if !ok {
			return
		}
		var req stocktakeItemHandleReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireEnum(req.Action, "action", stocktakeItemActions...),
				noteRequiredFor(req.Action, req.Note),
			)
		}) {
			return
		}
		note := strings.TrimSpace(req.Note)
		err := a.Asset.HandleStocktakeItem(c.Request.Context(), taskID, itemID, req.Action, note, httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "stocktake_item", fmt.Sprintf("%d/%d", taskID, itemID),
			map[string]any{"action": req.Action, "note": note})
		respond(c, apitypes.CodeOK, nil)
	}
}

// noteRequiredFor FIX/ESCALATE 必须留处理说明(CONFIRM 可空)。
func noteRequiredFor(action, note string) *httpx.ValidationError {
	if action == "FIX" || action == "ESCALATE" {
		return httpx.RequireString(note, "note", 255)
	}
	if strings.TrimSpace(note) == "" {
		return nil
	}
	return httpx.RequireString(note, "note", 255)
}
