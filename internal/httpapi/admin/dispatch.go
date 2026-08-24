package adminapi

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// registerDispatchRoutes 注册调度派单域路由(承接 order.yaml /dispatch 段)。
func registerDispatchRoutes(g *gin.RouterGroup, a *app.Application) {
	d := g.Group("/dispatch", requirePerm(a.User, "menu:dispatch"))
	// 跨区工单池:未指派(worker_id 空)的待派工单。
	d.GET("/pool", listDispatchPool(a))
	// 工单指派:候选师傅(masterId)回填工单。
	d.POST("/pool/:ticketNo/assign", assignDispatchTicket(a))
	// 我的工单:按师傅过滤。
	d.GET("/my-tickets", listMyDispatchTickets(a))
	// 改派/转单处理列表(全部改派台账)。
	d.GET("/transfers", listDispatchTransfers(a))
	// 工单转派:留痕改派台账 + 回填新师傅。
	d.POST("/tickets/:ticketNo/transfer", transferDispatchTicket(a))
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
			workerID, title, alert, pushdomain.ExtrasTicketNo(ticketNo))
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
