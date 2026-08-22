package workerapi

// W 师傅端门户工单转派/改约动作。

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

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
		tk, ok := workerOwnedTicketByNo(c, a, ticketNo)
		if !ok {
			return
		}
		// 目标闸门:在职 + 区域匹配(退回调度池 targetWorkerId=0 不校验)。
		var to *worker.Worker
		if req.TargetWorkerID != 0 {
			var ok bool
			if to, ok = workerTransferTargetOK(c, a, req.TargetWorkerID, tk); !ok {
				return
			}
		}
		if err := transferTicket(c, a, tk, to, &req); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// transferTicket 执行改派:目标师傅名回填;to nil = 退回池。
func transferTicket(c *gin.Context, a *app.Application, tk *order.DispatchTicket, to *worker.Worker, req *workerTransferReq) error {
	workerID, workerName := portalWorker(c)
	targetName := ""
	if to != nil {
		targetName = to.Name
	}
	if err := a.WorkOrder.AssignDispatchTicket(c.Request.Context(), tk.TicketNo, req.TargetWorkerID, targetName); err != nil {
		return err
	}
	notifyTicketTransferred(a, c, req.TargetWorkerID, targetName, tk.TicketNo)
	_, err := a.OrderLedger.AppendDispatchTransfer(c.Request.Context(), order.DispatchTransfer{
		TicketID: tk.TicketID, FromWorkerID: workerID, FromWorkerName: workerName,
		ToWorkerID: req.TargetWorkerID, ToWorkerName: targetName, Reason: req.Reason,
		OperatorAccountID: 0, // 师傅端自助改派,非账号操作
	})
	return err
}

// notifyTicketTransferred 师傅端转派成功后向目标师傅推通知(退回池 to=0 不推);
// 尽力而为,失败只留痕不影响改派。
func notifyTicketTransferred(a *app.Application, c *gin.Context, targetWorkerID int64, targetName, ticketNo string) {
	if a.PushNotifier == nil || targetWorkerID <= 0 {
		return
	}
	title, alert := pushdomain.TicketAssignedAlert(ticketNo, targetName)
	go func() {
		_ = a.PushNotifier.NotifyWorker(context.WithoutCancel(c.Request.Context()),
			targetWorkerID, title, alert, map[string]string{"ticketNo": ticketNo})
	}()
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
