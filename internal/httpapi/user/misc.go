package userapi

// 用户端门户 Misc 域:首页聚合/消息中心/优惠券/用量/自助排障/协议。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ---- Misc 域 ----

// registerPortalMiscRoutes Misc 域:首页聚合/消息/优惠券/地址/用量/自助排障/协议。
func registerPortalMiscRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/home", portalHome(a))
	g.GET("/messages", portalListMessages(a))
	g.POST("/messages/read-all", portalReadAllMessages(a))
	// 用户端 /coupons 仅挂在 /api/user/v1(与 admin /api/admin/v1 前缀隔离,无路由冲突)。
	g.GET("/coupons", portalListCoupons(a))
	g.GET("/addresses", portalListAddresses(a))
	g.POST("/addresses", portalCreateAddress(a))
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

// portalListCoupons GET /coupons?status=:我的优惠券(可从 userdata 券仓读取;未接入时降级静态)。
func portalListCoupons(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		status := c.DefaultQuery("status", "available")
		if a.UserData == nil {
			respond(c, apitypes.CodeOK, gin.H{
				"items": []gin.H{{
					"couponId": "C-001", "amount": 20.0, "threshold": 100.0,
					"title": "缴费满 100 减 20", "expireAt": "2026-12-31", "status": status,
				}}, "inviteLink": "https://u.ymm.example/invite",
			})
			return
		}
		rows, err := a.UserData.ListCoupons(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, r := range rows {
			if toInt64(r["customerId"]) != cid {
				continue
			}
			items = append(items, gin.H{
				"couponId": toStr(r["couponId"]), "name": toStr(r["name"]),
				"amount": float64(toInt64(r["amount"])) / 100, "threshold": 0.0,
				"title": toStr(r["name"]), "expireAt": toStr(r["expireAt"]),
				"status": couponStatus(toStr(r["status"]), status),
			})
		}
		invite := ""
		if cfg, err := a.UserData.GetInviteConfig(c.Request.Context()); err == nil && len(cfg) > 0 {
			invite = toStr(cfg[0]["inviteLink"])
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "inviteLink": invite})
	}
}

// couponStatus DB 券状态 → 契约状态(available/used/expired)。
func couponStatus(dbStatus, _ string) string {
	switch dbStatus {
	case "used", "expired":
		return dbStatus
	default:
		return "available"
	}
}

// portalListAddresses GET /addresses:我的家庭地址(契约 AddressInfo)。
func portalListAddresses(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		rows, err := a.UserData.ListUserAddresses(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, r := range rows {
			if toInt64(r["customerId"]) != cid {
				continue
			}
			items = append(items, gin.H{
				"addressId": toStr(r["id"]), "label": toStr(r["detail"]),
				"isDefault": r["isDefault"], "contact": toStr(r["contact"]),
				"phoneMasked": httpx.MaskPhone(toStr(r["phone"])),
				"community":   toStr(r["addrCode"]), "building": "", "door": "",
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalCreateAddress POST /addresses:新增家庭地址(user_addresses 落库)。
func portalCreateAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Community string `json:"community" binding:"required"`
			Building  string `json:"building"`
			Door      string `json:"door"`
			Contact   string `json:"contact" binding:"required"`
			Phone     string `json:"phone"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		phone := req.Phone
		if phone == "" {
			if cust, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
				phone = cust.Phone
			}
		}
		if _, err := a.UserData.CreateUserAddress(c.Request.Context(), udcustomer.UserAddress{
			CustomerID: cid, AddrCode: req.Community,
			Contact: req.Contact, Phone: phone,
			Detail:  req.Community + " " + req.Building + " " + req.Door,
			IsDefault: false,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalHome GET /home:客户信息 + 当前套餐 + 未缴合计 + 进行中订单 + 在用服务 + 消息红点。
func portalHome(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		name, phone, service := "用户", "", "服务在线 · 网络正常"
		if v, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
			name, phone = v.Name, httpx.MaskPhone(v.Phone)
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
		hasUnread, err := a.Portal.HasUnread(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		bal, err := a.Portal.Balance(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		plan := portalHomePlan(a, c, cid)
		respond(c, apitypes.CodeOK, gin.H{
			"customerName": name, "phoneMasked": phone, "onlineStatus": service,
			"hasUnread": hasUnread, "currentBill": due,
			"balance": bal, "plan": plan,
			"contractEnd": toStr(plan["contractEnd"]),
			"ongoingOrders": portalHomeOngoingOrders(a, c, cid),
			"services":     portalHomeServices(a, c),
		})
	}
}

// portalHomeOngoingOrders 进行中订单(PENDING/RESERVED/INSTALLING,契约 OrderSummary 字段)。
func portalHomeOngoingOrders(a *app.Application, c *gin.Context, cid int64) []gin.H {
	list, err := a.Order.List(c.Request.Context(), order.OrderQuery{CustomerID: cid})
	if err != nil {
		return []gin.H{}
	}
	items := make([]gin.H, 0, len(list))
	for _, item := range list {
		switch item.Status {
		case "PENDING", "RESERVED", "INSTALLING":
			items = append(items, gin.H{
				"orderNo": item.OrderNo, "status": item.Status,
				"statusLabel": portalOrderStatusLabel[item.Status],
				"productName": item.Product,
				"address":     addressName(a, c, item.AddressID, item.Address),
				"stage":       item.Stage, "stageLabel": portalOrderStageLabel(item.Stage),
				"estimateFinish": "", "canRate": false,
			})
		}
	}
	return items
}

// portalHomePlan 当前套餐(Home.plan),取 user_plans 行(ACTIVE 优先)。
func portalHomePlan(a *app.Application, c *gin.Context, cid int64) gin.H {
	plan := gin.H{"planId": "", "name": "", "monthlyFee": 0, "contractEnd": "",
		"status": "NONE", "installAddress": ""}
	if a.UserData == nil {
		return plan
	}
	rows, err := a.UserData.ListUserPlans(c.Request.Context())
	if err != nil {
		return plan
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) != cid {
			continue
		}
		pid := toInt64(r["productId"])
		fee := 0.0
		if p := portalProductByID(a, c, pid); p != nil {
			fee = p.MonthlyFee
		}
		status := toStr(r["status"])
		entry := gin.H{
			"planId": fmt.Sprintf("%d", pid), "name": toStr(r["planName"]),
			"monthlyFee": fee, "contractEnd": toStr(r["contractEnd"]),
			"status": status, "installAddress": portalDefaultAddress(a, c, cid),
		}
		if status == "ACTIVE" || toStr(plan["status"]) == "NONE" {
			plan = entry
		}
	}
	return plan
}

// portalHomeServices 已生效增值服务:ListAddonSubscriptions 中 subscribe 的增值包(契约 ServiceStatus,ACTIVE=已生效)。
func portalHomeServices(a *app.Application, c *gin.Context) []gin.H {
	services := []gin.H{}
	if a.UserData == nil {
		return services
	}
	views, err := portalAddonViews(a, c)
	if err != nil {
		return services
	}
	for _, v := range views {
		if sub, _ := v["subscribed"].(bool); !sub {
			continue
		}
		fee, _ := v["monthlyFee"].(float64)
		services = append(services, gin.H{
			"name": v["name"], "desc": portalHomeServiceDesc(fee),
			"status": "ACTIVE", "statusLabel": "已生效",
		})
	}
	return services
}

// portalHomeServiceDesc 服务卡描述:有月费显示 ¥/月,否则空(端上回退"查看套餐详情")。
func portalHomeServiceDesc(fee float64) string {
	if fee <= 0 {
		return ""
	}
	return fmt.Sprintf("¥%.0f/月", fee)
}

func portalListMessages(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		cat := c.DefaultQuery("category", "all")
		msgs, err := a.Portal.Messages(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, m := range msgs {
			if cat == "all" || m.Payload["category"] == cat {
				items = append(items, gin.H{"read": m.Read, "createdAt": m.CreatedAt, "payload": m.Payload})
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func portalReadAllMessages(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if err := a.Portal.MarkAllRead(c.Request.Context(), cid); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
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
