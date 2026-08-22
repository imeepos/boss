package workerapi

// W 师傅端门户工单域:首页/列表/详情/流转/任务池(worker/ticket.yaml)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/clock"
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
	return portalStatusOf(tk.Status, tk.WorkerID, workerID)
}

// portalStatusOf 状态映射核心(DispatchTicket/TicketItem 共用)。
func portalStatusOf(status string, ownerID, workerID int64) string {
	switch status {
	case "PENDING":
		return "TODO"
	case "DOING":
		if ownerID == workerID {
			return "DOING"
		}
		return "ACCEPTED"
	default: // DONE / CANCELED
		return "DONE"
	}
}

// portalItemStatus 列表项状态:订单终态兜底优先,再走工单状态映射。
func portalItemStatus(it order.TicketItem, workerID int64) string {
	if it.OrderStatus == "DONE" || it.OrderStatus == "CANCELED" {
		return "DONE"
	}
	return portalStatusOf(it.Status, it.WorkerID, workerID)
}

func workerOwnedTicket(c *gin.Context, tk *order.DispatchTicket) bool {
	workerID, _ := portalWorker(c)
	if tk.WorkerID == workerID {
		return true
	}
	respond(c, apitypes.CodeForbidden, nil)
	return false
}

// portalTicketStatusLabel 师傅端状态中文(fields.md 工单列表列)。
func portalTicketStatusLabel(s string) string {
	switch s {
	case "TODO":
		return "待领取"
	case "ACCEPTED":
		return "已接单"
	case "DOING":
		return "进行中"
	default:
		return "已完成"
	}
}

// portalTicketTypeOf 工单类型:报障单(complaints 联表有值)→ REPAIR/报障,否则 INSTALL/新装。
// 前端不再按 stages.length 推断(ISSUE.md worker 详情字段缺口)。
func portalTicketTypeOf(it order.TicketItem) (string, string) {
	if it.ComplaintType != "" || it.FaultTypeLabel != "" {
		return "REPAIR", "报障"
	}
	return "INSTALL", "新装"
}

// portalTicketOf 派单工单 → Ticket 视图(worker/schemas.yaml Ticket)。
// 地址/客户/环节取自列表读模型 TicketItem(联表订单),不再占位;
// 订单已终态(DONE/CANCELED)时工单视图按完成处理,避免"12/12 待领取"。
// 产品名/手机号脱敏一并随列表读模型填入,与详情对齐 OpenAPI TicketDetail。
func portalTicketOf(it order.TicketItem, workerID int64) gin.H {
	status := portalItemStatus(it, workerID)
	typ, typLabel := portalTicketTypeOf(it)
	return gin.H{
		"ticketNo": it.TicketNo, "bizNo": it.TicketNo, "type": typ,
		"typeLabel": typLabel, "statusLabel": portalTicketStatusLabel(status),
		"customerName":        it.CustomerName,
		"customerPhoneMasked": httpx.MaskPhone(it.CustomerPhone),
		"product":             it.OfferName,
		"address":             it.Address, "distanceKm": nil,
		"scheduleSlot": it.ScheduleSlot,
		"stage":        it.Stage, "stageTotal": 12, "status": status,
		"slaLeftMinutes": it.SlaLeftMinutes, "finishedAt": it.FinishedAt,
	}
}

// myTicketItems 取我的全部工单列表项(按当前师傅过滤)。
func myTicketItems(c *gin.Context, a *app.Application) ([]order.TicketItem, error) {
	workerID, _ := portalWorker(c)
	list, err := a.WorkOrder.ListTicketItems(c.Request.Context())
	if err != nil {
		return nil, err
	}
	out := make([]order.TicketItem, 0, len(list))
	for _, it := range list {
		if it.WorkerID == workerID {
			out = append(out, it)
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
		tickets, err := myTicketItems(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		today := gin.H{"accepted": 0, "finished": 0, "doing": 0, "todo": 0}
		ongoing := make([]gin.H, 0, len(tickets))
		for _, it := range tickets {
			s := portalItemStatus(it, workerID)
			incHomeCount(today, s)
			if s == "TODO" || s == "DOING" {
				ongoing = append(ongoing, portalTicketOf(it, workerID))
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
		tickets, err := myTicketItems(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		status := c.DefaultQuery("status", "all")
		items := make([]gin.H, 0, len(tickets))
		for _, it := range tickets {
			if status == "all" || portalItemStatus(it, workerID) == status {
				items = append(items, portalTicketOf(it, workerID))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerTicketsHistoryHandler 历史工单(近月维度,period 暂不分片)。
func workerTicketsHistoryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		workerID, _ := portalWorker(c)
		tickets, err := myTicketItems(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(tickets))
		for _, it := range tickets {
			if portalItemStatus(it, workerID) == "DONE" {
				items = append(items, portalTicketOf(it, workerID))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// workerTicketDetailHandler 工单详情:订单 12 环节时间轴 + 四码对照 + 风控位。
//
// 字段对齐 api/openapi/worker/schemas.yaml::TicketDetail。
// 工单头信息(客户/地址/产品/分光器端口/预绑定标签/预约时间/报障/完成时间)经由
// a.WorkOrder.GetTicketItemByNo 联表获取;手机号走 httpx.MaskPhone 脱敏出门;
// OpenAPI 预留位(分光器/预绑定/预约时间/报障/SLA/远程诊断)后端暂无数据源,返回空,
// 前端按空值不渲染处理。
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
		respond(c, apitypes.CodeOK, workerTicketDetailPayload(c, a, tk, currentWorkerID))
	}
}

// workerTicketDetailPayload 详情视图聚合(订单环节 + 工单头 + 四码 + 风控)。
func workerTicketDetailPayload(c *gin.Context, a *app.Application, tk *order.DispatchTicket, currentWorkerID int64) gin.H {
	ord, stages, err := a.Order.Track(c.Request.Context(), tk.OrderID)
	if err != nil {
		respondErr(c, err)
		return gin.H{}
	}
	item, err := a.WorkOrder.GetTicketItemByNo(c.Request.Context(), tk.TicketNo)
	if err != nil {
		respondErr(c, err)
		return gin.H{}
	}
	status := portalTicketStatus(*tk, currentWorkerID)
	typ, typLabel := portalTicketTypeOf(*item)
	payload := gin.H{
		"ticketNo": tk.TicketNo, "bizNo": ord.OrderNo,
		"status": status, "statusLabel": portalTicketStatusLabel(status),
		"type": typ, "typeLabel": typLabel, "distanceKm": nil,
		// 已有结构
		"stages":    portalStages(stages),
		"quad":      portalQuadH(a, c, ord.AddressID),
		"riskCheck": gin.H{"blacklistHit": false, "graylistHit": false},
	}
	for k, v := range ticketHeadFields(item) {
		payload[k] = v
	}
	return payload
}

// ticketHeadFields 工单头字段视图(对齐 OpenAPI TicketDetail)。
func ticketHeadFields(item *order.TicketItem) gin.H {
	return gin.H{
		"product":             item.OfferName,
		"customerName":        item.CustomerName,
		"customerPhoneMasked": httpx.MaskPhone(item.CustomerPhone),
		"address":             item.Address,
		"splitterPort":        item.SplitterPort,
		"preBindTag":          item.PreBindTag,
		"scheduleSlot":        item.ScheduleSlot,
		"faultTypeLabel":      item.FaultTypeLabel,
		"reportedAt":          item.ReportedAt,
		"slaLeftMinutes":      item.SlaLeftMinutes,
		"remoteDiagnosis":     item.RemoteDiagnosis,
		"finishedAt":          item.FinishedAt,
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
			item["finishedAt"] = l.FinishedAt.In(clock.Location()).Format("01-02 15:04")
		}
		out = append(out, item)
	}
	return out
}
