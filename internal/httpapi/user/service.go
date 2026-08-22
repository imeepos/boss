package userapi

// 用户端门户客服域:报修/投诉/智能客服/常见问题。

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
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