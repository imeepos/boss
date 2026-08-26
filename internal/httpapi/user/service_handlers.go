package userapi

// 用户端门户客服域:报修 handler + 智能客服 + 常见问题。
// 投诉与建议 handler 拆见 complaint_handlers.go;service.go 仅留路由表。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

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
				items = append(items, faultItem(f))
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// faultItem 报修项视图(列表与详情共用)。
func faultItem(f order.Complaint) gin.H {
	return gin.H{
		"ticketNo": f.TicketNo, "type": f.Type, "typeLabel": portalFaultTypeLabelFromStored(f.Type),
		"status": f.Status, "statusLabel": portalComplaintStatusLabel[f.Status],
	}
}

// portalCreateFault POST /faults:报障入客服工单域(complaints 落库)+ 站内消息。
func portalCreateFault(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req portalFaultReq
		if !httpx.BindBody(c, &req) {
			return
		}
		ticketNo, ok := portalCreateFaultFlow(c, a, cid, req)
		if !ok {
			return
		}
		respond(c, apitypes.CodeOK, faultCreatedPayload(ticketNo, req))
	}
}

// portalFaultReq 报障请求体。
type portalFaultReq struct {
	FaultType   string `json:"faultType" binding:"required,oneof=no_internet slow ont_fault other"`
	Address     string `json:"address" binding:"required"`
	Description string `json:"description" binding:"required"`
	Contact     string `json:"contact"`
}

// faultCreatedPayload 报障受理回执。
func faultCreatedPayload(ticketNo string, req portalFaultReq) gin.H {
	return gin.H{
		"ticketNo": ticketNo, "faultType": req.FaultType,
		"faultTypeLabel": portalFaultTypeLabel[req.FaultType],
		"address":        req.Address, "createdAt": time.Now(), "status": "OPEN", "statusLabel": "受理中",
	}
}

// portalCreateFaultTicket 落报修工单到 complaints。
func portalCreateFaultTicket(c *gin.Context, a *app.Application, cid, legID int64, legName string,
	req struct {
		FaultType   string `json:"faultType" binding:"required,oneof=no_internet slow ont_fault other"`
		Address     string `json:"address" binding:"required"`
		Description string `json:"description" binding:"required"`
		Contact     string `json:"contact"`
	},
) (string, error) {
	ticketNo, err := a.Portal.NextNo(c.Request.Context(), "TKT")
	if err != nil {
		return "", err
	}
	if _, err := a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
		TicketNo: ticketNo, CustomerID: cid, LegalEntityID: legID, LegalEntityName: legName,
		Type: portalFaultTypeName(req.FaultType), Status: "OPEN",
	}); err != nil {
		return "", err
	}
	return ticketNo, nil
}

// portalCreateFaultMessage 写报修受理站内消息。
func portalCreateFaultMessage(c *gin.Context, a *app.Application, cid int64, ticketNo, desc string) error {
	msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
	if err != nil {
		return err
	}
	return a.Portal.PutMessage(c.Request.Context(), cid, gin.H{
		"messageId": msgID, "category": "fault", "title": "报修已受理",
		"content": desc, "tag": "报修", "tagLevel": "fault",
		"createdAt": time.Now(), "read": false,
	})
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
				respond(c, apitypes.CodeOK, gin.H{"fault": faultItem(f), "sla": "≤4h", "timeline": []gin.H{
					{"step": 1, "title": "提交报修", "result": "DONE"},
					{"step": 2, "title": "受理派单", "result": "DOING"},
				}})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalCreateFaultFlow 报障三步:法人主体 → 建投诉工单 → 首条沟通消息;
// 失败已回写响应。
func portalCreateFaultFlow(c *gin.Context, a *app.Application, cid int64, req portalFaultReq) (string, bool) {
	legID, legName, err := portalComplaintMeta(c.Request.Context(), a, cid)
	if err != nil {
		respondErr(c, err)
		return "", false
	}
	ticketNo, err := portalCreateFaultTicket(c, a, cid, legID, legName, req)
	if err != nil {
		respondErr(c, err)
		return "", false
	}
	if err := portalCreateFaultMessage(c, a, cid, ticketNo, req.Description); err != nil {
		respondErr(c, err)
		return "", false
	}
	return ticketNo, true
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