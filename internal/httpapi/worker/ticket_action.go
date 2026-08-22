package workerapi

// W 师傅端门户工单动作:领取/签到/导航/回退/重试。
// 转单/改约在 ticket_action_transfer.go,投诉/修复上报在 ticket_action_complaint.go。

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerTicketActions 工单动作路由。
func registerWorkerTicketActions(g *gin.RouterGroup, a *app.Application) {
	g.POST("/tickets/:ticketNo/accept", func(c *gin.Context) {
		assignTicketToMe(c, a, c.Param("ticketNo"), false)
	})
	g.POST("/tickets/:ticketNo/checkin", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"checkedInAt": nowHM()})
	})
	g.GET("/tickets/:ticketNo/navi", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{
			"address": "", "distanceKm": 0, "etaMinutes": 0, "entrance": "",
		})
	})
	g.POST("/tickets/:ticketNo/transfer", workerTransferHandler(a))
	g.POST("/tickets/:ticketNo/reschedule", workerRescheduleHandler(a))
	g.POST("/tickets/:ticketNo/rollback", workerRollbackHandler(a))
	g.POST("/tickets/:ticketNo/retry", workerRetryHandler(a))
	g.POST("/tickets/:ticketNo/complaint", workerComplaintHandler(a))
	g.POST("/tickets/:ticketNo/repair-report", workerRepairReportHandler(a))
}

// assignTicketToMe 领取/抢单:PENDING 才可抢(accept 容忍已指派给自己)。
func assignTicketToMe(c *gin.Context, a *app.Application, ticketNo string, grab bool) {
	tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
	if err != nil {
		respondErr(c, err)
		return
	}
	workerID, workerName := portalWorker(c)
	// 资格闸门:在职 + 负责区域匹配 + 接单设置(在线/接单类型)。
	if !workerMayAccept(c, a, workerID, tk) {
		return
	}
	if !grab && tk.WorkerID != workerID {
		respond(c, apitypes.CodeForbidden, nil)
		return
	}
	if tk.Status != "PENDING" || (grab && tk.WorkerID != 0) {
		respond(c, apitypes.CodeStateInvalid, nil)
		return
	}
	var assignErr error
	if grab {
		assignErr = a.WorkOrder.AssignPendingDispatchTicket(c.Request.Context(), ticketNo, workerID, workerName)
	} else {
		assignErr = a.WorkOrder.ClaimDispatchTicket(c.Request.Context(), ticketNo, workerID, workerName)
	}
	if err := assignErr; err != nil {
		// 并发领取:预检通过但抢占落空,按状态无效而非不存在。
		if errors.Is(err, order.ErrOrderNotFound) {
			respond(c, apitypes.CodeStateInvalid, nil)
			return
		}
		respondErr(c, err)
		return
	}
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

// workerAuditOK 无独立落表的动作(消息已读/扫码异常/签到):审计留痕 + OK。
func workerAuditOK(a *app.Application, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", c.Param("ticketNo"),
			map[string]any{"action": action})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerRollbackHandler 回退上一环节:真实落库(删最新环节日志、stage 前移、
// status 逆向迁移、DONE 工单随动 DOING),前端刷新时间轴即可见。
func workerRollbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		if err := a.Order.RollbackStage(c.Request.Context(), tk.OrderID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", tk.TicketNo,
			map[string]any{"action": "rollback"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerRetryHandler 重试当前失败环节:命中订单激活回调则 retries+1。
func workerRetryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerOwnedTicket(c, tk) {
			return
		}
		list, err := a.OrderLedger.ListActivationCallbacks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, cb := range list {
			if cb.OrderID == tk.OrderID && cb.Result == "FAILED" {
				if err := a.OrderLedger.RetryActivationCallback(c.Request.Context(), cb.ID); err != nil {
					respondErr(c, err)
					return
				}
				respond(c, apitypes.CodeOK, gin.H{"ok": true})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}
