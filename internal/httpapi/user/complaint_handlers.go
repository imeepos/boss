package userapi

// 用户端门户 — 投诉与建议 handler:列表(分页)/详情/提交。
// 与 service_handlers.go(报修+智能客服)解耦,各自 ≤300 行。

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cs"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalComplaintPageSize 用户端投诉列表默认 pageSize;与 ProfileApi.addresses /
// OrderApi.list 口径一致,客户端在用户代理上能稳定推断。page 夹紧到 [1,∞)。
const (
	portalComplaintPageSize = 10
	portalComplaintMaxSize  = 50
)

// portalListComplaints GET /complaints:我的投诉列表(分页)。query 接受 page/pageSize,
// 默认 page=1 pageSize=10;服务端内部夹紧 pageSize≤50 防滥用。
func portalListComplaints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		page, _ := strconv.Atoi(c.Query("page"))
		size, _ := strconv.Atoi(c.Query("pageSize"))
		if page < 1 {
			page = 1
		}
		if size < 1 {
			size = portalComplaintPageSize
		}
		if size > portalComplaintMaxSize {
			size = portalComplaintMaxSize
		}
		list, hasMore, err := a.WorkOrder.ListComplaintsByCustomerPaged(c.Request.Context(), cid, page, size)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		for _, cp := range list {
			items = append(items, complaintItem(cp))
		}
		respond(c, apitypes.CodeOK, gin.H{
			"items": items, "page": page, "pageSize": size, "hasMore": hasMore,
		})
	}
}

// complaintItem 投诉项视图。description/contact/relOrderNo 来自迁移 000151 新增列,
// 空串表示历史数据未填,前端按空态渲染。
func complaintItem(cp order.Complaint) gin.H {
	t := strings.TrimPrefix(cp.Type, "用户投诉: ")
	return gin.H{
		"complaintId": cp.TicketNo,
		"type":        t,
		"typeLabel":   portalComplaintTypeLabel[t],
		"relOrderNo":  cp.RelOrderNo,
		"description": cp.Description,
		"contact":     cp.Contact,
		"status":      cp.Status,
		"statusLabel": portalComplaintStatusLabel[cp.Status],
		"createdAt":   cp.CreatedAt,
		"slaDeadline": cp.SlaDeadline,
		"remoteDiag":  cp.RemoteDiagnosis,
		"closedAt":    formatClosedAt(cp.ClosedAt),
	}
}

// portalComplaintDetail GET /complaints/:ticketNo:我的投诉详情(含处理历史 timeline)。
// timeline 由 cs_ticket_events 派生(状态变更/催单/升级);空 timeline 表示
// 工单还在 OPEN 状态尚未触发事件,前端按空态渲染。
func portalComplaintDetail(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		ticketNo := c.Param("ticketNo")
		cp, err := a.WorkOrder.GetComplaintByNoAndCustomer(c.Request.Context(), ticketNo, cid)
		if err != nil {
			if err == order.ErrOrderNotFound {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		var events []cs.TicketEvent
		if a.Tickets != nil {
			events, _ = a.Tickets.ListTicketEventsByNo(c.Request.Context(), ticketNo)
		}
		timeline := make([]gin.H, 0, len(events)+1)
		timeline = append(timeline, gin.H{
			"step": 1, "title": "提交投诉", "result": "DONE",
			"meta": cp.CreatedAt, "eventType": "SUBMITTED",
		})
		for _, e := range events {
			timeline = append(timeline, gin.H{
				"step":       len(timeline) + 1,
				"title":      complaintEventTitle(e),
				"result":     "DONE",
				"meta":       formatEventTime(e.CreatedAt),
				"eventType":  e.EventType,
				"fromStatus": e.FromStatus,
				"toStatus":   e.ToStatus,
				"note":       e.Note,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{
			"complaint": complaintItem(*cp),
			"timeline":  timeline,
		})
	}
}

// portalCreateComplaint POST /complaints:投诉入客服工单域。
// 客户端 type 取值 attitude/quality/billing/suggestion/other,服务端落库前缀
// "用户投诉: " 以便与管理后台工单域(type=SINGLE_OUTAGE/...)区隔;
// description/contact/relOrderNo 真实落库(迁移 000151),详情/列表可回显。
func portalCreateComplaint(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Type        string `json:"type" binding:"required,oneof=attitude quality billing suggestion other"`
			Description string `json:"description" binding:"required,min=4,max=500"`
			Contact     string `json:"contact" binding:"omitempty,max=32"`
			RelOrderNo  string `json:"relOrderNo" binding:"omitempty,max=64"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if !portalCreateComplaintFlow(c, a, cid, req) {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalComplaintMeta 取客户归属运营主体(complaints.legal_entity 非空约束)。
// 被报障与投诉两个 flow 共用,放在 complaint_handlers.go。
func portalComplaintMeta(ctx context.Context, a *app.Application, cid int64) (int64, string, error) {
	cu, err := a.Customer.Get(ctx, cid)
	if err != nil {
		return 0, "", err
	}
	entities, err := a.User.ListLegalEntities(ctx)
	if err != nil {
		return 0, "", err
	}
	for _, e := range entities {
		if e.ID == cu.LegalEntityID {
			return cu.LegalEntityID, e.Name, nil
		}
	}
	return cu.LegalEntityID, "", nil
}

// portalCreateComplaintFlow 投诉建档:法人主体 → 派单号 → 投诉工单(OPEN)。
// description/contact/relOrderNo 一并落库(迁移 000151);失败已回写响应。
func portalCreateComplaintFlow(c *gin.Context, a *app.Application, cid int64, req struct {
	Type        string `json:"type" binding:"required,oneof=attitude quality billing suggestion other"`
	Description string `json:"description" binding:"required,min=4,max=500"`
	Contact     string `json:"contact" binding:"omitempty,max=32"`
	RelOrderNo  string `json:"relOrderNo" binding:"omitempty,max=64"`
}) bool {
	legID, legName, err := portalComplaintMeta(c.Request.Context(), a, cid)
	if err != nil {
		respondErr(c, err)
		return false
	}
	ticketNo, err := a.Portal.NextNo(c.Request.Context(), "TKT")
	if err != nil {
		respondErr(c, err)
		return false
	}
	_, err = a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
		TicketNo: ticketNo, CustomerID: cid, LegalEntityID: legID, LegalEntityName: legName,
		Type: "用户投诉: " + req.Type, Status: "OPEN",
		Description: req.Description, Contact: req.Contact, RelOrderNo: req.RelOrderNo,
	})
	if err != nil {
		respondErr(c, err)
		return false
	}
	return true
}

// complaintEventTitle 事件类型 → 中文标签(详情页时间轴展示用)。
func complaintEventTitle(e cs.TicketEvent) string {
	switch e.EventType {
	case "STATUS_CHANGED":
		return "状态变更 · " + portalComplaintStatusLabel[e.ToStatus]
	case "ESCALATED":
		return "投诉升级"
	default:
		return e.EventType
	}
}

func formatClosedAt(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

func formatEventTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}