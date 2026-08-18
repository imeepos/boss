package app

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerDispatchRoutes 注册调度派单域路由(承接 order.yaml /dispatch 段)。
func registerDispatchRoutes(g *gin.RouterGroup, a *Application) {
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
		var req assignTicketReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		w, err := a.Worker.GetWorker(c.Request.Context(), req.MasterID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.WorkOrder.AssignDispatchTicket(c.Request.Context(), c.Param("ticketNo"), w.ID, w.Name); err != nil {
			respondErr(c, err)
			return
		}
		a.recordAudit(c, "dispatch.assign", "dispatch_ticket", c.Param("ticketNo"),
			map[string]any{"masterId": req.MasterID})
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
		var req transferTicketReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		ticket, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		to, err := a.Worker.GetWorker(c.Request.Context(), req.ToMasterID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.appendTransfer(c, ticket, to.ID, to.Name, req.Reason); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

// appendTransfer 追加改派台账并把工单指到新师傅。
func (a *Application) appendTransfer(c *gin.Context, ticket *order.DispatchTicket, toID int64, toName, reason string) error {
	if _, err := a.OrderLedger.AppendDispatchTransfer(c.Request.Context(), order.DispatchTransfer{
		TicketID: ticket.TicketID, FromWorkerID: ticket.WorkerID, FromWorkerName: ticket.WorkerName,
		ToWorkerID: toID, ToWorkerName: toName, Reason: reason,
	}); err != nil {
		return err
	}
	if err := a.WorkOrder.AssignDispatchTicket(c.Request.Context(), ticket.TicketNo, toID, toName); err != nil {
		return err
	}
	a.recordAudit(c, "dispatch.transfer", "dispatch_ticket", ticket.TicketNo,
		map[string]any{"toMasterId": toID, "reason": reason})
	return nil
}

// assignTicketReq 工单指派请求体(masterId 必填)。
type assignTicketReq struct {
	MasterID int64 `json:"masterId" binding:"required"`
}

// transferTicketReq 工单转派请求体(toMasterId/reason 必填)。
type transferTicketReq struct {
	ToMasterID int64  `json:"toMasterId" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
}
