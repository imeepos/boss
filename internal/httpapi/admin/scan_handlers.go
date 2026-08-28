package adminapi

// W5 扫码闭环路由具名 handler(承接 registerScanRoutes 扁平路由表)。
// worker 扫码绑定/拆机扫码;admin 扫码日志/四码对账/冲突处理/孤儿清理。
// 请求体类型 scanBindReq 与工具 workerFromClaims 见 scan.go。

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// scanBindHandler POST /tickets/{ticketNo}/scan-bind:扫码绑定(MATCH 才推进环节9)。
func scanBindHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req scanBindReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireTicketInScope(c, a, tk) {
			return
		}
		if !scanVerifyAndAdvance(c, a, tk, req) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"result": "MATCH"})
	}
}

// scanDismantleHandler POST /tickets/{ticketNo}/dismantle/scan:拆机扫码核验。
func scanDismantleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req scanBindReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !requireTicketInScope(c, a, tk) {
			return
		}
		if err := a.QuadLink.UnbindRequireScan(c.Request.Context(), tk.OrderID, req.EPC); err != nil {
			httpx.RespondScanErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "quadlink", c.Param("ticketNo"), map[string]any{"action": "dismantle-scan"})
		respond(c, apitypes.CodeOK, nil)
	}
}

// scanLogsHandler GET /scan-logs:扫码日志(按 orderId 过滤)。
func scanLogsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkOrder.ListScanLogs(c.Request.Context(), queryInt64(c, "orderId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// quadConflictsListHandler GET /quad-conflicts:四码对账冲突列表(过滤 status=CONFLICT)。
func quadConflictsListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.QuadLink.ListLinks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		out := make([]quadlink.QuadLink, 0, len(list))
		for _, it := range list {
			if it.Status == "CONFLICT" {
				out = append(out, it)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": out})
	}
}

// quadConflictResolveHandler POST /quad-conflicts/{id}/resolve:标记冲突已处理。
func quadConflictResolveHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		linkID, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.QuadLink.ResolveConflict(c.Request.Context(), linkID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "quadlink", c.Param("id"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// quadLinksReconcileHandler POST /quad-links/reconcile:四码对账(返回冲突统计)。
func quadLinksReconcileHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rep, err := a.QuadLink.Reconcile(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "quadlink", "reconcile", map[string]any{"conflict": rep.Conflict})
		respond(c, apitypes.CodeOK, rep)
	}
}

// quadLinksPurgeOrphansHandler POST /quad-links/purge-orphans:清理孤儿四码绑定。
func quadLinksPurgeOrphansHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := a.QuadLink.PurgeOrphans(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "quadlink", "purge-orphans", map[string]any{"deleted": n})
		respond(c, apitypes.CodeOK, gin.H{"deleted": n})
	}
}

// scanBindWorker 扫码绑定执行师傅:JWT 无账号时回退工单档案。
func scanBindWorker(c *gin.Context, tk *order.DispatchTicket) (int64, string) {
	workerID, workerName := workerFromClaims(c)
	if workerID == 0 { // 工单未带师傅且 JWT 无账号:回退工单档案
		workerID, workerName = tk.WorkerID, tk.WorkerName
	}
	return workerID, workerName
}

// scanVerifyAndAdvance 四码核对;核对一致(MATCH)才推进环节9;失败已回写响应。
func scanVerifyAndAdvance(c *gin.Context, a *app.Application, tk *order.DispatchTicket, req scanBindReq) bool {
	workerID, workerName := scanBindWorker(c, tk)
	result, err := a.QuadLink.VerifyScan(c.Request.Context(), quadlink.ScanReq{
		OrderID: tk.OrderID, WorkerID: workerID, WorkerName: workerName,
		ScannedEPC: req.EPC, OfflineCalc: req.Offline,
	})
	if err != nil {
		httpx.RespondScanErr(c, err)
		return false
	}
	if result != "MATCH" { // 核对不一致不推进,直接回结果
		respond(c, apitypes.CodeOK, gin.H{"result": result})
		return false
	}
	if err := a.Order.ScanBind(c.Request.Context(), tk.OrderID); err != nil {
		// 幂等重放:环节9 已完成(重复扫码)按成功回 MATCH,不报错。
		if errors.Is(err, order.ErrIllegalTransition) {
			respond(c, apitypes.CodeOK, gin.H{"result": "MATCH"})
			return false
		}
		respondErr(c, err)
		return false
	}
	return true
}
