package app

// 用户端门户客服域:报修/投诉/智能客服/常见问题。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalServiceRoutes 客服域:报修/投诉/智能客服/常见问题。
func registerPortalServiceRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/faults", portalListFaults)
	g.POST("/faults", portalCreateFault)
	g.GET("/faults/:ticketNo", portalFaultDetail)
	g.POST("/complaints", portalCreateComplaint(a))
	g.POST("/service/chat", portalChat)
	g.GET("/service/faq", portalFaq)
}

// portalFaults 报修进程内台账(TKT 单号);客服工单域对接见报告。
var portalFaults = map[int64][]gin.H{}

func portalListFaults(c *gin.Context) {
	cid, _ := requireCustomer(c)
	respond(c, apitypes.CodeOK, gin.H{"items": portalFaults[cid]})
}

// portalCreateFault POST /faults:生成报修单 + 站内消息。
func portalCreateFault(c *gin.Context) {
	cid, _ := requireCustomer(c)
	var req struct {
		FaultType   string `json:"faultType" binding:"required,oneof=no_internet slow ont_fault other"`
		Address     string `json:"address" binding:"required"`
		Description string `json:"description" binding:"required"`
		Contact     string `json:"contact"`
	}
	if !bindBody(c, &req) {
		return
	}
	f := gin.H{
		"ticketNo": portal.payNo(), "faultType": req.FaultType,
		"faultTypeLabel": map[string]string{"no_internet": "无法上网", "slow": "网速慢",
			"ont_fault": "光猫故障", "other": "其他"}[req.FaultType],
		"address": req.Address, "createdAt": time.Now(), "status": "OPEN", "statusLabel": "受理中",
	}
	portalFaults[cid] = append([]gin.H{f}, portalFaults[cid]...)
	portal.putMessage(cid, gin.H{
		"messageId": portal.payNo(), "category": "fault", "title": "报修已受理",
		"content": req.Description, "tag": "报修", "tagLevel": "fault",
		"createdAt": time.Now(), "read": false,
	})
	respond(c, apitypes.CodeOK, f)
}

func portalFaultDetail(c *gin.Context) {
	cid, _ := requireCustomer(c)
	for _, f := range portalFaults[cid] {
		if f["ticketNo"] == c.Param("ticketNo") {
			respond(c, apitypes.CodeOK, gin.H{"fault": f, "sla": "≤4h", "timeline": []gin.H{
				{"step": 1, "title": "提交报修", "result": "DONE"},
				{"step": 2, "title": "受理派单", "result": "DOING"},
			}})
			return
		}
	}
	respond(c, apitypes.CodeNotFound, nil)
}

// portalCreateComplaint POST /complaints:同步入客服工单域(admin 投诉队列可见)。
func portalCreateComplaint(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Type        string `json:"type" binding:"required"`
			Description string `json:"description" binding:"required"`
			RelOrderNo  string `json:"relOrderNo"`
		}
		if !bindBody(c, &req) {
			return
		}
		_, err := a.WorkOrder.CreateComplaint(c.Request.Context(), order.Complaint{
			CustomerID: cid, Type: "用户投诉: " + req.Type, Status: "OPEN",
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
	if !bindBody(c, &req) {
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
