package app

// 用户端门户 Misc 域:首页聚合/消息中心/优惠券/用量/自助排障/协议。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// ---- Misc 域 ----

// registerPortalMiscRoutes Misc 域:首页聚合/消息/优惠券/用量/自助排障/协议。
func registerPortalMiscRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/home", portalHome(a))
	g.GET("/messages", portalListMessages)
	g.POST("/messages/read-all", portalReadAllMessages)
	// TODO(契约冲突待商议): GET /api/v1/coupons 已由 admin userdata 占用(gin 静态路由不允许同路径双注册)。
	// 用户端与 admin 共用 /api/v1 前缀下的重叠端点需网关分流或独立前缀裁决,见对账报告。
	// g.GET("/coupons", portalCoupons)
	g.GET("/usage", portalUsage)
	g.GET("/diy/steps", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"items": portalDiySections})
	})
	g.GET("/agreement", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{
			"userAgreement": []string{"服务条款", "费用与账期", "终止与违约"},
			"privacyPolicy": []string{"信息收集范围", "使用与共享", "保存期限与删除"},
		})
	})
}

// portalHome GET /home:客户信息 + 未缴合计 + 消息红点(账单/订单详情聚合见报告)。
func portalHome(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		name, phone, service := "用户", "", "服务在线 · 网络正常"
		if v, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
			name, phone = v.Name, maskPhone(v.Phone)
			if v.ServiceStatus != "ACTIVE" {
				service = "服务状态: " + v.ServiceStatus
			}
		}
		var due float64
		if bills, err := a.Billing.ListBills(c.Request.Context(), cid); err == nil {
			for _, b := range bills {
				if b.Status != "PAID" {
					due += b.Amount
				}
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"customerName": name, "phoneMasked": phone, "onlineStatus": service,
			"hasUnread": portal.hasUnread(cid), "currentBill": due,
			"balance": portal.balance(cid), "ongoingOrders": []any{}, "services": []any{},
		})
	}
}

func portalListMessages(c *gin.Context) {
	cid, _ := requireCustomer(c)
	_ = cid
	cat := c.DefaultQuery("category", "all")
	items := make([]gin.H, 0)
	for _, m := range portal.msgs(cid) {
		if cat == "all" || m["category"] == cat {
			items = append(items, m)
		}
	}
	respond(c, apitypes.CodeOK, gin.H{"items": items})
}

func portalReadAllMessages(c *gin.Context) {
	cid, _ := requireCustomer(c)
	portal.markAllRead(cid)
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

func portalCoupons(c *gin.Context) {
	status := c.DefaultQuery("status", "available")
	respond(c, apitypes.CodeOK, gin.H{
		"items": []gin.H{{
			"couponId": "C-001", "amount": 20.0, "threshold": 100.0,
			"title": "缴费满 100 减 20", "expireAt": "2026-12-31", "status": status,
		}},
		"inviteLink": "https://portal.example/invite",
	})
}

func portalUsage(c *gin.Context) {
	period := c.DefaultQuery("period", "current")
	quota, used := 1024.0, 356.8
	respond(c, apitypes.CodeOK, gin.H{
		"periodLabel": map[string]string{"current": "本账期", "last": "上一账期", "six_month": "近 6 月"}[period],
		"used":        used, "unit": "GB", "quota": quota, "percent": int(used / quota * 100),
		"dailyAvg": 11.9, "forecastRemain": 667.2,
		"detail": gin.H{"down": 300.2, "up": 56.6, "iptvNote": "IPTV 用量不计入宽带配额"},
	})
}

// portalDiySections 自助排障静态引导(契约 DiySection)。
var portalDiySections = []gin.H{
	{"id": "power", "title": "设备检查", "desc": "先确认光猫与路由器电源",
		"steps": []gin.H{{"title": "查看指示灯", "desc": "LOS 红灯常亮表示线路故障"}}},
	{"id": "wifi", "title": "Wi-Fi 检查", "desc": "排除无线干扰",
		"steps": []gin.H{{"title": "靠近路由器测试", "desc": "排除距离与穿墙衰减"}}},
	{"id": "cable", "title": "线路检查", "desc": "确认入户光纤与网线",
		"steps": []gin.H{{"title": "重新插拔光纤接头", "desc": "注意接头清洁"}}},
}
