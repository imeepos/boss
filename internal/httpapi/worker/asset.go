package workerapi

// W 师傅端门户资产域(worker/asset.yaml):路由表 + 共享类型/视图工具。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/worker"
)

// registerWorkerPortalAssetRoutes 资产域路由(wauth 组)。
func registerWorkerPortalAssetRoutes(g *gin.RouterGroup, a *app.Application) {
	g.POST("/tickets/:ticketNo/dismantle/scan", workerDismantleScanHandler(a))
	g.GET("/tickets/:ticketNo/replace", workerReplaceGetHandler(a))
	g.POST("/tickets/:ticketNo/replace", workerReplacePostHandler(a))
	g.GET("/replacements", workerReplacementsHandler(a))
	g.POST("/replacements/:id/complete", workerReplacementCompleteHandler(a))
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

// workerReplaceReq 换件提交请求体(契约 asset.yaml Replace)。
type workerReplaceReq struct {
	OldEpc string `json:"oldEpc"`
	NewEpc string `json:"newEpc"`
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

// workerAppendReplaceLog 落换件流水(师傅 ID + 工单 + EPC + 时间)。
func workerAppendReplaceLog(c *gin.Context, a *app.Application, tk *order.DispatchTicket, req workerReplaceReq) error {
	workerID, _ := portalWorker(c)
	_, err := a.WorkerEvent.AppendReplaceLog(c.Request.Context(), worker.ReplaceLog{
		WorkerID: workerID, DispatchTicketID: tk.TicketID, TicketNo: tk.TicketNo,
		OldEpc: req.OldEpc, NewEpc: req.NewEpc, CreatedAt: time.Now(),
	})
	return err
}
