package workerapi

// W 师傅端门户工单动作:领取/签到/导航/转单/改约/回退/重试/投诉/修复上报。

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
	g.POST("/tickets/:ticketNo/rollback", workerAuditOK(a, "rollback"))
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

// workerTransferReq 转单请求;targetWorkerId 空=退回调度池。
type workerTransferReq struct {
	Reason         string `json:"reason" binding:"required"`
	TargetWorkerID int64  `json:"targetWorkerId"`
	Remark         string `json:"remark"`
}

// workerTransferHandler 转单/改派:改派留台账(dispatch_transfers)。
func workerTransferHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerTransferReq
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
		if err := transferTicket(c, a, tk, &req); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// transferTicket 执行改派:目标师傅名回填;目标 0=退回池。
func transferTicket(c *gin.Context, a *app.Application, tk *order.DispatchTicket, req *workerTransferReq) error {
	workerID, workerName := portalWorker(c)
	targetName := ""
	if req.TargetWorkerID != 0 {
		w, err := a.Worker.GetWorker(c.Request.Context(), req.TargetWorkerID)
		if err != nil {
			return err
		}
		targetName = w.Name
	}
	if err := a.WorkOrder.AssignDispatchTicket(c.Request.Context(), tk.TicketNo, req.TargetWorkerID, targetName); err != nil {
		return err
	}
	_, err := a.OrderLedger.AppendDispatchTransfer(c.Request.Context(), order.DispatchTransfer{
		TicketID: tk.TicketID, FromWorkerID: workerID, FromWorkerName: workerName,
		ToWorkerID: req.TargetWorkerID, ToWorkerName: targetName, Reason: req.Reason,
		OperatorAccountID: 0, // 师傅端自助改派,非账号操作
	})
	return err
}

// workerRescheduleReq 改约请求体(对齐 OpenAPI /tickets/{ticketNo}/reschedule)。
type workerRescheduleReq struct {
	NewDate string `json:"newDate" binding:"required"` // 如 08-22
	NewSlot string `json:"newSlot" binding:"required"` // 如 14:00-16:00
	Reason  string `json:"reason"`
	Remark  string `json:"remark"`
}

// workerRescheduleHandler 改约:落表 dispatch_tickets.schedule_slot + 审计留痕。
// 格式 "MM-DD HH:MM-HH:MM"(newDate + " " + newSlot)。
func workerRescheduleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerRescheduleReq
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
		scheduleSlot := req.NewDate + " " + req.NewSlot
		if err := a.WorkOrder.UpdateScheduleSlot(c.Request.Context(), tk.TicketNo, scheduleSlot); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "reschedule", "dispatch_ticket", tk.TicketNo,
			map[string]any{"scheduleSlot": scheduleSlot, "reason": req.Reason, "remark": req.Remark})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "scheduleSlot": scheduleSlot})
	}
}

// workerAuditOK 无独立落表的动作(回退):审计留痕 + OK(缺口见报告)。
func workerAuditOK(a *app.Application, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.RecordAudit(a, c, "状态变更", "worker_ticket", c.Param("ticketNo"),
			map[string]any{"action": action})
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

// workerComplaintReq 现场投诉登记。
type workerComplaintReq struct {
	Category string `json:"category" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

// workerComplaintHandler 投诉登记:转客服域(complaints,status=OPEN)。
func workerComplaintHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerComplaintReq
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
		_, err = a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
			TicketNo: tk.TicketNo, OrderID: tk.OrderID,
			LegalEntityID: tk.LegalEntityID, LegalEntityName: tk.LegalEntityName,
			Type: req.Category, Status: "OPEN",
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// workerRepairReportReq 修复结果上报。
type workerRepairReportReq struct {
	Result string `json:"result"`
	Remark string `json:"remark"`
}

// workerRepairReportHandler 修复上报:FIXED → 报障 CLOSED,UNFIXED → PROCESSING。
func workerRepairReportHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req workerRepairReportReq
		if !httpx.BindAndValidate(c, &req, func() error {
			if req.Result != "FIXED" && req.Result != "UNFIXED" {
				return &httpx.ValidationError{Field: "result", Message: "must be FIXED or UNFIXED"}
			}
			return nil
		}) {
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
		if req.Result == "FIXED" {
			if err := a.WorkOrder.CloseComplaint(c.Request.Context(), c.Param("ticketNo")); err != nil {
				respond(c, apitypes.CodeOK, gin.H{"reviewPassed": false, "status": "PROCESSING"})
				return
			}
			respond(c, apitypes.CodeOK, gin.H{"reviewPassed": true, "status": "CLOSED"})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"reviewPassed": false, "status": "PROCESSING", "remark": req.Remark})
	}
}
