package workerapi

// W 师傅端门户工单域:首页/列表/详情/流转/任务池(worker/ticket.yaml)。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalStageNames 新装 12 环节名(terms.md 第 1 节,禁止增删改序)。
var portalStageNames = [13]string{
	"", "下单", "资源核查", "端口预占", "合同收费", "标签预绑定", "创建账号",
	"预下发配置", "派单", "扫码绑定", "激活", "激活回调", "更新GIS",
}

// portalTicketStatus 派单工单状态 → 师傅端状态(TODO/ACCEPTED/DOING/DONE)。
func portalTicketStatus(tk order.DispatchTicket, workerID int64) string {
	switch tk.Status {
	case "PENDING":
		return "TODO"
	case "DOING":
		if tk.WorkerID == workerID {
			return "DOING"
		}
		return "ACCEPTED"
	default: // DONE / CANCELED
		return "DONE"
	}
}

// portalTicketOf 派单工单 → Ticket 视图(worker/schemas.yaml Ticket)。
func portalTicketOf(tk order.DispatchTicket, workerID int64) gin.H {
	status := portalTicketStatus(tk, workerID)
	return gin.H{
		"ticketNo": tk.TicketNo, "bizNo": tk.TicketNo, "type": "INSTALL",
		"address": "", "distanceKm": 0, "scheduleSlot": "",
		"stage": 9, "stageTotal": 12, "status": status,
		"slaLeftMinutes": nil, "finishedAt": "",
	}
}

// myTickets 取我的全部工单(按当前师傅过滤)。
func myTickets(c *gin.Context, a *app.Application) ([]order.DispatchTicket, error) {
	workerID, _ := portalWorker(c)
	list, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
	if err != nil {
		return nil, err
	}
	out := make([]order.DispatchTicket, 0, len(list))
	for _, tk := range list {
		if tk.WorkerID == workerID {
			out = append(out, tk)
		}
	}
	return out, nil
}

// registerWorkerPortalTicketRoutes 工单域路由(wauth 组)。
func registerWorkerPortalTicketRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/home", workerHomeHandler(a))
	g.GET("/tickets", workerTicketsHandler(a))
	g.GET("/tickets/history", workerTicketsHistoryHandler(a))
	g.GET("/tickets/:ticketNo", workerTicketDetailHandler(a))
	registerWorkerTicketActions(g, a)
	g.GET("/hall", workerHallHandler(a))
	g.POST("/hall/:ticketNo/grab", workerGrabHandler(a))
}

// workerHomeHandler 首页:师傅档案 + 今日计数 + 进行中工单。
func workerHomeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		w, err := a.Worker.GetWorker(c.Request.Context(), workerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		tickets, err := myTickets(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		today := gin.H{"accepted": 0, "finished": 0, "doing": 0, "todo": 0}
		ongoing := make([]gin.H, 0, len(tickets))
		for _, tk := range tickets {
			s := portalTicketStatus(tk, workerID)
			incHomeCount(today, s)
			if s == "TODO" || s == "DOING" {
				ongoing = append(ongoing, portalTicketOf(tk, workerID))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"workerName": w.Name, "groupName": "", "phoneMasked": httpx.MaskPhone(w.Phone),
			"today": today, "ongoing": ongoing,
		})
	}
}

// incHomeCount 按师傅端状态累加首页计数。
func incHomeCount(today gin.H, s string) {
	if v, ok := today[s].(int); ok {
		today[s] = v + 1
	}
}

// maskPhone 手机号脱敏(138****1234)。
// workerTicketsHandler 我的工单列表(status doing/todo/done/all)。
func workerTicketsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		tickets, err := myTickets(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		status := c.DefaultQuery("status", "all")
		items := make([]gin.H, 0, len(tickets))
		for _, tk := range tickets {
			if status == "all" || portalTicketStatus(tk, workerID) == status {
				items = append(items, portalTicketOf(tk, workerID))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerTicketsHistoryHandler 历史工单(近月维度,period 暂不分片)。
func workerTicketsHistoryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		tickets, err := myTickets(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(tickets))
		for _, tk := range tickets {
			if tk.Status == "DONE" || tk.Status == "CANCELED" {
				items = append(items, portalTicketOf(tk, workerID))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerTicketDetailHandler 工单详情:订单 12 环节时间轴 + 四码对照 + 风控位。
func workerTicketDetailHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		tk, err := a.WorkOrder.GetDispatchTicketByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		currentWorkerID, _ := portalWorker(c)
		if tk.WorkerID != currentWorkerID {
			respond(c, apitypes.CodeForbidden, nil)
			return
		}
		ord, stages, err := a.Order.Track(c.Request.Context(), tk.OrderID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"ticketNo": tk.TicketNo, "bizNo": ord.OrderNo,
			"status": portalTicketStatus(*tk, currentWorkerID), "statusLabel": "",
			"stages": portalStages(stages), "quad": portalQuadH(a, c, ord.AddressID),
			"riskCheck": gin.H{"blacklistHit": false, "graylistHit": false},
		})
	}
}

// portalStages 环节日志 → StageItem 视图(环节名取 terms.md 12 环节)。
func portalStages(logs []order.StageLog) []gin.H {
	out := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		name := ""
		if int(l.Stage) < len(portalStageNames) {
			name = portalStageNames[l.Stage]
		}
		item := gin.H{"stage": l.Stage, "name": name, "result": l.Result, "note": ""}
		if l.FinishedAt != nil {
			item["finishedAt"] = l.FinishedAt.Format("01-02 15:04")
		}
		out = append(out, item)
	}
	return out
}

// workerHallHandler 任务池:未指派的公开工单。
func workerHallHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.WorkOrder.ListDispatchTickets(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, tk := range list {
			if tk.WorkerID == 0 && tk.Status == "PENDING" {
				items = append(items, portalTicketOf(tk, 0))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerGrabHandler 抢单:先到先得,PENDING 且未指派才可抢。
func workerGrabHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignTicketToMe(c, a, c.Param("ticketNo"), true)
	}
}

// nowHM 当前时间 HH:mm(签到/打卡回执)。
func nowHM() string { return time.Now().Format("15:04") }
