package adminapi

// 调度派单域 handler 实现(从 dispatch.go 抽出,registerDispatchRoutes 只剩扁平路由表)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// listDispatchPool 跨区工单池:未指派的待派工单。
func listDispatchPool(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// assignDispatchTicket 工单指派:候选师傅回填工单。
func assignDispatchTicket(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// listMyDispatchTickets 我的工单:按师傅过滤。
func listMyDispatchTickets(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// listDispatchTransfers 改派/转单处理列表。
func listDispatchTransfers(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.OrderLedger.ListDispatchTransfers(c.Request.Context(), 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// transferDispatchTicket 工单转派:留痕改派台账 + 回填新师傅。
func transferDispatchTicket(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
