package userapi

// 门户首页聚合 /home(客户信息 + 宽带卡 + 进行中订单 + 增值服务)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalHome GET /home:客户信息 + 当前套餐 + 未缴合计 + 进行中订单 + 在用服务 + 消息红点。
func portalHome(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		name, phone, service := portalHomeCustomer(a, c, cid)
		due := portalHomeDue(c, a, cid)
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
			"contractEnd":   toStr(plan["contractEnd"]),
			"ongoingOrders": portalHomeOngoingOrders(a, c, cid),
			"services":      portalHomeServices(a, c),
		})
	}
}

// portalHomeCustomer 取客户名/脱敏手机号/在线状态文案;Customer 取不到则给默认值。
func portalHomeCustomer(a *app.Application, c *gin.Context, cid int64) (string, string, string) {
	name, phone, service := "用户", "", "服务在线 · 网络正常"
	v, err := a.Customer.Get(c.Request.Context(), cid)
	if err != nil {
		return name, phone, service
	}
	name, phone = v.Name, httpx.MaskPhone(v.Phone)
	if v.ServiceStatus != "ACTIVE" {
		service = "服务状态: " + v.ServiceStatus
	}
	return name, phone, service
}

// portalHomeDue 客户未缴账单合计(ListBills 失败按 0)。
func portalHomeDue(c *gin.Context, a *app.Application, cid int64) float64 {
	bills, err := a.Billing.ListBills(c.Request.Context(), cid)
	if err != nil {
		return 0
	}
	var due float64
	for _, b := range bills {
		if b.Status != "PAID" {
			due += b.Amount
		}
	}
	return due
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
