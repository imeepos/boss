package workerapi

// W 师傅端门户工单动作:领取/签到/导航/回退/重试。
// 转单/改约在 ticket_action_transfer.go,投诉/修复上报在 ticket_action_complaint.go。

import (
	"errors"
	"log"

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
	if !assignEligible(c, a, workerID, tk, grab) {
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

// assignEligible 领取资格闸门:在职 + 区域匹配 + 接单设置,accept 须已指派自己,
// 状态须 PENDING(抢单时还须未指派);失败已回写响应。
func assignEligible(c *gin.Context, a *app.Application, workerID int64, tk *order.DispatchTicket, grab bool) bool {
	if !workerMayAccept(c, a, workerID, tk) {
		return false
	}
	if !grab && tk.WorkerID != workerID {
		respond(c, apitypes.CodeForbidden, nil)
		return false
	}
	if tk.Status != "PENDING" || (grab && tk.WorkerID != 0) {
		respond(c, apitypes.CodeStateInvalid, nil)
		return false
	}
	return true
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
// 假成功守卫:RollbackStage 回执 after==before 即 0 行生效,显性 5xx +
// [fake-success] ALERT 留痕——回归史:rollback 曾只审计不落库仍返回成功。
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
		before, after, err := a.Order.RollbackStage(c.Request.Context(), tk.OrderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if after == before {
			log.Printf("[fake-success] ALERT rollback no-effect ticket=%s order=%d stage=%d",
				tk.TicketNo, tk.OrderID, before)
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", tk.TicketNo,
			map[string]any{"action": "rollback", "before": before, "after": after})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "before": before, "after": after})
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
