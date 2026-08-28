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
		w, _, ok := assignResolve(c, a, ticketNo, req)
		if !ok {
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

// assignResolve 指派前置校验链:师傅可用 → 工单存在 → 跨区闸门;失败已回写响应。
func assignResolve(c *gin.Context, a *app.Application, ticketNo string, req assignTicketReq) (*worker.Worker, *order.DispatchTicket, bool) {
	w, err := a.Worker.GetWorker(c.Request.Context(), req.MasterID)
	if err != nil {
		respondErr(c, err)
		return nil, nil, false
	}
	if !workerAssignable(w) {
		respond(c, apitypes.CodeInvalidParam, gin.H{"error": "worker not active"})
		return nil, nil, false
	}
	ticket, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
	if err != nil {
		respondErr(c, err)
		return nil, nil, false
	}
	// 跨区指派:默认拒绝并回 40900 提醒,前端二次确认后带 force 强派。
	if ticket != nil && !worker.RegionMatched(w.RegionID, ticket.RegionID) && !req.Force {
		respondRegionMismatch(c, ticket, w.RegionID)
		return nil, nil, false
	}
	return w, ticket, true
}

// respondRegionMismatch 跨区指派/转派的 40900 提醒(forceRequired 前端二次确认)。
func respondRegionMismatch(c *gin.Context, ticket *order.DispatchTicket, workerRegionID int64) {
	respond(c, apitypes.CodeConflict, gin.H{
		"error": "region mismatch", "forceRequired": true,
		"ticketRegionId": ticket.RegionID, "ticketRegionName": ticket.RegionName,
		"workerRegionId": workerRegionID,
	})
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
		ticket, to, ok := transferResolve(c, a, ticketNo, req)
		if !ok {
			return
		}
		if err := appendTransfer(a, c, ticket, to.ID, to.Name, req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// transferResolve 转派前置校验链:工单可转 → 目标师傅可用 → 非当前指派 → 跨区闸门;
// 失败已回写响应。
func transferResolve(c *gin.Context, a *app.Application, ticketNo string, req transferTicketReq) (*order.DispatchTicket, *worker.Worker, bool) {
	ticket, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), ticketNo)
	if err != nil {
		respondErr(c, err)
		return nil, nil, false
	}
	if !requireTicketInScope(c, a, ticket) {
		return nil, nil, false
	}
	// 终态工单不可转派;转派目标即当前师傅视为无效操作。
	if ticket.Status == "DONE" || ticket.Status == "CANCELED" {
		respond(c, apitypes.CodeInvalidParam, gin.H{"error": "ticket already " + ticket.Status})
		return nil, nil, false
	}
	to, err := a.Worker.GetWorker(c.Request.Context(), req.ToMasterID)
	if err != nil {
		respondErr(c, err)
		return nil, nil, false
	}
	if !workerAssignable(to) {
		respond(c, apitypes.CodeInvalidParam, gin.H{"error": "worker not active"})
		return nil, nil, false
	}
	if to.ID == ticket.WorkerID {
		respond(c, apitypes.CodeInvalidParam, gin.H{"error": "target worker is current assignee"})
		return nil, nil, false
	}
	// 跨区转派:同指派,默认 40900 提醒,force 确认后放行。
	if !worker.RegionMatched(to.RegionID, ticket.RegionID) && !req.Force {
		respondRegionMismatch(c, ticket, to.RegionID)
		return nil, nil, false
	}
	return ticket, to, true
}
