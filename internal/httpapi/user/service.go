package userapi

// 用户端门户客服域:报修/投诉/智能客服/常见问题。

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalServiceRoutes 客服域:报修/投诉/智能客服/常见问题。
func registerPortalServiceRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/faults", portalListFaults(a))
	g.POST("/faults", portalCreateFault(a))
	g.GET("/faults/:ticketNo", portalFaultDetail(a))
	g.POST("/faults/:ticketNo/urge", portalUrgeFault(a))
	g.GET("/faults/:ticketNo/contact", portalFaultContact(a))
	g.GET("/complaints", portalListComplaints(a))
	g.POST("/complaints", portalCreateComplaint(a))
	g.POST("/service/chat", portalChat)
	g.GET("/service/faq", portalFaq)
}

// portalComplaintTypeLabel 投诉类型 → 中文标签。
var portalComplaintTypeLabel = map[string]string{
	"attitude": "服务态度", "quality": "服务质量", "billing": "计费问题",
	"suggestion": "意见建议", "other": "其他",
}

// portalListComplaints GET /complaints:我的投诉列表(工单域 complaints 按客户过滤)。
func portalListComplaints(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		list, err := a.WorkOrder.ListComplaints(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, cp := range list {
			if cp.CustomerID != cid {
				continue
			}
			items = append(items, gin.H{
				"complaintId": cp.TicketNo, "type": strings.TrimPrefix(cp.Type, "用户投诉: "),
				"typeLabel":  portalComplaintTypeLabel[strings.TrimPrefix(cp.Type, "用户投诉: ")],
				"relOrderNo": "", "description": "", "status": cp.Status,
				"statusLabel": portalComplaintStatusLabel[cp.Status], "createdAt": "",
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalComplaintStatusLabel 投诉状态 → 中文标签。
var portalComplaintStatusLabel = map[string]string{
	"OPEN": "受理中", "PROCESSING": "处理中", "CLOSED": "已关闭",
}

// portalFaultTypeLabel 报障类型 → 中文标签。
var portalFaultTypeLabel = map[string]string{
	"no_internet": "无法上网", "slow": "网速慢", "ont_fault": "光猫故障", "other": "其他",
}

// portalFaultTypeName 报障类型 → 工单域 type 值(admin 队列口径)。
func portalFaultTypeName(t string) string { return "用户报障: " + t }

func portalFaultTypeLabelFromStored(t string) string {
	const prefix = "用户报障: "
	if strings.HasPrefix(t, prefix) {
		t = strings.TrimPrefix(t, prefix)
	}
	return portalFaultTypeLabel[t]
}

// portalComplaintMeta 取客户归属运营主体(complaints.legal_entity 非空约束)。
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

// portalListFaults GET /faults:我的报修列表(工单域 complaints 按客户过滤)。
func portalListFaults(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		list, err := a.WorkOrder.ListComplaints(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, f := range list {
			if f.CustomerID == cid {
				items = append(items, gin.H{
					"ticketNo": f.TicketNo, "type": f.Type, "typeLabel": portalFaultTypeLabelFromStored(f.Type),
					"status": f.Status, "statusLabel": portalComplaintStatusLabel[f.Status],
				})
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalCreateFault POST /faults:报障入客服工单域(complaints 落库)+ 站内消息。
func portalCreateFault(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			FaultType   string `json:"faultType" binding:"required,oneof=no_internet slow ont_fault other"`
			Address     string `json:"address" binding:"required"`
			Description string `json:"description" binding:"required"`
			Contact     string `json:"contact"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		legID, legName, err := portalComplaintMeta(c.Request.Context(), a, cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		ticketNo, err := a.Portal.NextNo(c.Request.Context(), "TKT")
		if err != nil {
			respondErr(c, err)
			return
		}
		if _, err := a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
			TicketNo: ticketNo, CustomerID: cid, LegalEntityID: legID, LegalEntityName: legName,
			Type: portalFaultTypeName(req.FaultType), Status: "OPEN",
		}); err != nil {
			respondErr(c, err)
			return
		}
		msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Portal.PutMessage(c.Request.Context(), cid, gin.H{
			"messageId": msgID, "category": "fault", "title": "报修已受理",
			"content": req.Description, "tag": "报修", "tagLevel": "fault",
			"createdAt": time.Now(), "read": false,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"ticketNo": ticketNo, "faultType": req.FaultType,
			"faultTypeLabel": portalFaultTypeLabel[req.FaultType],
			"address":        req.Address, "createdAt": time.Now(), "status": "OPEN", "statusLabel": "受理中",
		})
	}
}

// portalFaultDetail GET /faults/:ticketNo:我的报修详情(工单域寻址,归属校验)。
func portalFaultDetail(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		list, err := a.WorkOrder.ListComplaints(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, f := range list {
			if f.TicketNo == c.Param("ticketNo") && f.CustomerID == cid {
				respond(c, apitypes.CodeOK, gin.H{"fault": gin.H{
					"ticketNo": f.TicketNo, "type": f.Type, "typeLabel": portalFaultTypeLabelFromStored(f.Type),
					"status": f.Status, "statusLabel": portalComplaintStatusLabel[f.Status],
				}, "sla": "≤4h", "timeline": []gin.H{
					{"step": 1, "title": "提交报修", "result": "DONE"},
					{"step": 2, "title": "受理派单", "result": "DOING"},
				}})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

func portalCreateComplaint(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Type        string `json:"type" binding:"required"`
			Description string `json:"description" binding:"required"`
			RelOrderNo  string `json:"relOrderNo"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		legID, legName, err := portalComplaintMeta(c.Request.Context(), a, cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		ticketNo, err := a.Portal.NextNo(c.Request.Context(), "TKT")
		if err != nil {
			respondErr(c, err)
			return
		}
		_, err = a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
			TicketNo: ticketNo, CustomerID: cid, LegalEntityID: legID, LegalEntityName: legName,
			Type: "用户投诉: " + req.Type, Status: "OPEN",
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalChat POST /service/chat:规则应答(转接 AI 网关见报告)。
func portalChat(c *gin.Context) {
	var req struct {
		Message string `json:"message" binding:"required"`
	}
	if !httpx.BindBody(c, &req) {
		return
	}
	reply, toHuman := "您可先尝试「自助排障」引导;未解决将转人工客服。", false
	if len(req.Message) > 50 {
		reply, toHuman = "已为您转接人工客服。", true
	}
	respond(c, apitypes.CodeOK, gin.H{"reply": reply, "toHuman": toHuman})
}

// portalFaq GET /service/faq 静态常见问题。
func portalFaq(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{"items": []gin.H{
		{"id": "f1", "question": "如何修改密码?", "answer": "我的-账号安全-修改密码。"},
		{"id": "f2", "question": "账单多久出一次?", "answer": "每月 1 日出上一账期账单。"},
	}})
}
