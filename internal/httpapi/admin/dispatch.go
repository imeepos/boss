package adminapi

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

// registerDispatchRoutes 注册调度派单域路由(承接 order.yaml /dispatch 段)。
func registerDispatchRoutes(g *gin.RouterGroup, a *app.Application) {
	d := g.Group("/dispatch", requirePerm(a.User, "menu:dispatch"))

	// 跨区工单池:未指派(worker_id 空)的待派工单。
	d.GET("/pool", func(c *gin.Context) {
		tickets, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		pool := make([]order.DispatchTicket, 0, len(tickets))
		for _, t := range tickets {
			if t.WorkerID == 0 {
				pool = append(pool, t)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": pool})
	})

	// 工单指派:候选师傅(masterId)回填工单。
	d.POST("/pool/:ticketNo/assign", func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req assignTicketReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		w, err := a.Worker.GetWorker(c.Request.Context(), req.MasterID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerAssignable(w) {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "worker not active"})
			return
		}
		ticket, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		// 跨区指派:默认拒绝并回 40900 提醒,前端二次确认后带 force 强派。
		if ticket != nil && !worker.RegionMatched(w.RegionID, ticket.RegionID) && !req.Force {
			respond(c, apitypes.CodeConflict, gin.H{
				"error": "region mismatch", "forceRequired": true,
				"ticketRegionId": ticket.RegionID, "ticketRegionName": ticket.RegionName,
				"workerRegionId": w.RegionID,
			})
			return
		}
		if err := a.WorkOrder.AssignDispatchTicket(c.Request.Context(), ticketNo, w.ID, w.Name,
			order.AssignOpt{ScheduleSlot: req.ScheduleSlot, PreBindTag: req.PreBindTag}); err != nil {
			respondErr(c, err)
			return
		}
		notifyTicketAssigned(a, c, w.ID, w.Name, ticketNo)
		httpx.RecordAudit(a, c, "dispatch.assign", "dispatch_ticket", ticketNo,
			map[string]any{"masterId": req.MasterID, "scheduleSlot": req.ScheduleSlot, "preBindTag": req.PreBindTag})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 我的工单:按师傅过滤。
	d.GET("/my-tickets", func(c *gin.Context) {
		workerID := queryInt64(c, "workerId")
		tickets, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		mine := make([]order.DispatchTicket, 0, len(tickets))
		for _, t := range tickets {
			if t.WorkerID == workerID {
				mine = append(mine, t)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": mine})
	})

	// 改派/转单处理列表(全部改派台账)。
	d.GET("/transfers", func(c *gin.Context) {
		list, err := a.OrderLedger.ListDispatchTransfers(c.Request.Context(), 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 工单转派:留痕改派台账 + 回填新师傅。
	d.POST("/tickets/:ticketNo/transfer", func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		if ticketNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticketNo is required"})
			return
		}
		var req transferTicketReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		ticket, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		// 终态工单不可转派;转派目标即当前师傅视为无效操作。
		if ticket.Status == "DONE" || ticket.Status == "CANCELED" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticket already " + ticket.Status})
			return
		}
		to, err := a.Worker.GetWorker(c.Request.Context(), req.ToMasterID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !workerAssignable(to) {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "worker not active"})
			return
		}
		if to.ID == ticket.WorkerID {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "target worker is current assignee"})
			return
		}
		// 跨区转派:同指派,默认 40900 提醒,force 确认后放行。
		if !worker.RegionMatched(to.RegionID, ticket.RegionID) && !req.Force {
			respond(c, apitypes.CodeConflict, gin.H{
				"error": "region mismatch", "forceRequired": true,
				"ticketRegionId": ticket.RegionID, "ticketRegionName": ticket.RegionName,
				"workerRegionId": to.RegionID,
			})
			return
		}
		if err := appendTransfer(a, c, ticket, to.ID, to.Name, req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

// workerAssignable 指派/转派目标师傅必须在职(status=1 且未离职):离职师傅不可接单。
func workerAssignable(w *worker.Worker) bool {
	return w != nil && w.Status == 1 && w.LeftAt == nil
}

// appendTransfer 追加改派台账并把工单指到新师傅。
func appendTransfer(a *app.Application, c *gin.Context, ticket *order.DispatchTicket, toID int64, toName, reason string) error {
	if _, err := a.OrderLedger.AppendDispatchTransfer(c.Request.Context(), order.DispatchTransfer{
		TicketID: ticket.TicketID, FromWorkerID: ticket.WorkerID, FromWorkerName: ticket.WorkerName,
		ToWorkerID: toID, ToWorkerName: toName, Reason: reason,
	}); err != nil {
		return err
	}
	if err := a.WorkOrder.AssignDispatchTicket(c.Request.Context(), ticket.TicketNo, toID, toName); err != nil {
		return err
	}
	notifyTicketAssigned(a, c, toID, toName, ticket.TicketNo)
	httpx.RecordAudit(a, c, "dispatch.transfer", "dispatch_ticket", ticket.TicketNo,
		map[string]any{"toMasterId": toID, "reason": reason})
	return nil
}

// notifyTicketAssigned 派单/转派成功后向目标师傅推通知;尽力而为,失败只留痕不影响派单。
func notifyTicketAssigned(a *app.Application, c *gin.Context, workerID int64, workerName, ticketNo string) {
	if a.PushNotifier == nil || workerID <= 0 {
		return
	}
	title, alert := pushdomain.TicketAssignedAlert(ticketNo, workerName)
	go func() {
		_ = a.PushNotifier.NotifyWorker(context.WithoutCancel(c.Request.Context()),
			workerID, title, alert, map[string]string{"ticketNo": ticketNo})
	}()
}

// assignTicketReq 工单指派请求体(masterId 必填;scheduleSlot/preBindTag 可选)。
type assignTicketReq struct {
	MasterID     int64  `json:"masterId" binding:"required"`
	ScheduleSlot string `json:"scheduleSlot"` // 预约时间段，如 08-22 14:00-16:00
	PreBindTag   string `json:"preBindTag"`   // 预绑定 EPC 标签，如 EPC-0001
	Force        bool   `json:"force"`        // 跨区指派二次确认:true=已知晓跨区仍强制派单
}

// transferTicketReq 工单转派请求体(toMasterId/reason 必填)。
type transferTicketReq struct {
	ToMasterID int64  `json:"toMasterId" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
	Force      bool   `json:"force"` // 跨区转派二次确认
}
