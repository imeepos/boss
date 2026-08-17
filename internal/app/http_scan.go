package app

// W5 扫码闭环 handler:worker 扫码绑定/拆机扫码 + admin 扫码日志/四码对账/冲突处理。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ticket 扫码请求体(worker/scan.yaml、worker/asset.yaml)。
type scanBindReq struct {
	EPC     string `json:"epc" binding:"required"`
	Offline bool   `json:"offline"`
}

// respondScanErr 扫码域错误 → 统一错误码(不一致 40920 / 未预绑定 40910 / 未扫码 42200)。
func respondScanErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, quadlink.ErrScanMismatch):
		respond(c, apitypes.CodeScanMismatch, nil)
	case errors.Is(err, quadlink.ErrNotPrebound):
		respond(c, apitypes.CodeStateInvalid, nil)
	case errors.Is(err, quadlink.ErrScanRequired):
		respond(c, apitypes.CodeInvalidParam, nil)
	default:
		respondErr(c, err)
	}
}

// workerFromClaims 从 JWT 取操作师傅(账号ID + 用户名)。
func workerFromClaims(c *gin.Context) (int64, string) {
	if v, ok := c.Get(middleware.CtxClaims); ok {
		if claims, ok := v.(*auth.Claims); ok {
			return claims.AccountID, claims.Username
		}
	}
	return 0, ""
}

// registerScanRoutes 注册 worker 扫码闭环 + admin 四码对账路由。
func registerScanRoutes(g *gin.RouterGroup, a *Application) {
	w := g.Group("/tickets")
	w.POST("/:ticketNo/scan-bind", func(c *gin.Context) {
		var req scanBindReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		workerID, workerName := workerFromClaims(c)
		if workerID == 0 { // 工单未带师傅且 JWT 无账号:回退工单档案
			workerID, workerName = tk.WorkerID, tk.WorkerName
		}
		result, err := a.QuadLink.VerifyScan(c.Request.Context(), quadlink.ScanReq{
			OrderID: tk.OrderID, WorkerID: workerID, WorkerName: workerName,
			ScannedEPC: req.EPC, OfflineCalc: req.Offline,
		})
		if err != nil {
			respondScanErr(c, err)
			return
		}
		if result == "MATCH" { // 核对一致才推进环节9
			if err := a.Order.ScanBind(c.Request.Context(), tk.OrderID); err != nil {
				respondErr(c, err)
				return
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"result": result})
	})

	w.POST("/:ticketNo/dismantle/scan", func(c *gin.Context) {
		var req scanBindReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.QuadLink.UnbindRequireScan(c.Request.Context(), tk.OrderID, req.EPC); err != nil {
			respondScanErr(c, err)
			return
		}
		a.recordAudit(c, "状态变更", "quadlink", c.Param("ticketNo"), map[string]any{"action": "dismantle-scan"})
		respond(c, apitypes.CodeOK, nil)
	})

	q := g.Group("", requirePerm(a.User, "menu:quadlink"))
	q.GET("/scan-logs", func(c *gin.Context) {
		list, err := a.WorkOrder.ListScanLogs(c.Request.Context(), queryInt64(c, "orderId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	q.GET("/quad-conflicts", func(c *gin.Context) {
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
	})
	q.POST("/quad-conflicts/:id/resolve", func(c *gin.Context) {
		linkID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.QuadLink.ResolveConflict(c.Request.Context(), linkID); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "状态变更", "quadlink", c.Param("id"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	q.POST("/quad-links/reconcile", func(c *gin.Context) {
		rep, err := a.QuadLink.Reconcile(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "数据变更", "quadlink", "reconcile", map[string]any{"conflict": rep.Conflict})
		respond(c, apitypes.CodeOK, rep)
	})
}
