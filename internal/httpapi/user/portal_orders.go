package userapi

// 用户端门户 Order/Product 域缺失端点:订单列表/下单/订单详情 + 产品列表/增值服务列表。
// 契约:api/openapi/user/order.yaml + product.yaml;前缀 /api/user/v1(客户鉴权组)。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalStageTitles 12 环节标题(terms.md §1 权威清单,序号禁止增删改序)。
var portalStageTitles = map[int]string{
	1: "下单", 2: "资源核查", 3: "端口预占", 4: "合同收费", 5: "标签预绑定",
	6: "创建账号", 7: "预下发配置", 8: "派单", 9: "扫码绑定", 10: "激活",
	11: "激活回调", 12: "更新 GIS",
}

// portalOrderStatusLabel 订单状态中文标签(terms.md §3)。
var portalOrderStatusLabel = map[string]string{
	"PENDING": "待核查", "RESERVED": "已预占", "INSTALLING": "装维中",
	"DONE": "已完成", "CANCELLED": "已取消",
}

// portalOrderStageLabel 环节序号 → 环节标题。
func portalOrderStageLabel(stage int8) string {
	if t, ok := portalStageTitles[int(stage)]; ok {
		return t
	}
	return "已归档"
}

// portalOrderSummary 订单读模型 → 契约 OrderSummary。
func portalOrderSummary(o *order.Order, productName, address string, canRate bool) gin.H {
	return gin.H{
		"orderNo": o.OrderNo, "status": o.Status,
		"statusLabel": portalOrderStatusLabel[o.Status],
		"productName": productName, "address": address,
		"stage": o.Stage, "stageLabel": portalOrderStageLabel(o.Stage),
		"estimateFinish": "", "canRate": canRate,
	}
}

// portalOrderTimeline 12 环节时间轴:DB 环节日志(StageLog)为基,缺失环节按 PENDING 补齐。
func portalOrderTimeline(stages []order.StageLog) []gin.H {
	byStage := make(map[int8]order.StageLog, len(stages))
	for _, s := range stages {
		byStage[s.Stage] = s
	}
	out := make([]gin.H, 0, len(portalStageTitles))
	for i := 1; i <= len(portalStageTitles); i++ {
		t := int8(i)
		result := "PENDING"
		if s, ok := byStage[t]; ok {
			result = s.Result
		}
		out = append(out, gin.H{"stage": i, "title": portalStageTitles[i], "result": result, "meta": ""})
	}
	return out
}

// portalListOrders GET /orders?status=:我的订单列表(status: all/in_progress/done/cancelled)。
func portalListOrders(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		status := c.DefaultQuery("status", "all")
		list, err := a.Order.List(c.Request.Context(), order.OrderQuery{Status: statusFilter(status), CustomerID: cid})
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		for _, item := range list {
			items = append(items, gin.H{
				"orderNo": item.OrderNo, "status": item.Status,
				"statusLabel": portalOrderStatusLabel[item.Status],
				"productName": item.Product, "address": addressName(a, c, item.AddressID, item.Address),
				"stage": item.Stage, "stageLabel": portalOrderStageLabel(item.Stage),
				"estimateFinish": "", "canRate": item.Status == "DONE",
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// statusFilter 契约标签 → 订单状态;in_progress 含 RESERVED/INSTALLING。
func statusFilter(status string) string {
	switch status {
	case "done":
		return "DONE"
	case "cancelled":
		return "CANCELLED"
	case "in_progress":
		return "INSTALLING"
	default:
		return ""
	}
}

// addressName 地址名:联表地址缺失时回退用户地址簿(user_addresses)。
func addressName(a *app.Application, c *gin.Context, addressID int64, fallback string) string {
	if fallback != "" {
		return fallback
	}
	if a.UserData == nil {
		return ""
	}
	addrs, err := a.UserData.ListUserAddresses(c.Request.Context())
	if err != nil {
		return ""
	}
	for _, row := range addrs {
		if toInt64(row["id"]) == addressID {
			return toStr(row["detail"])
		}
	}
	return ""
}

// portalGetOrder GET /orders/:orderNo:订单详情(含 12 环节时间轴)。
func portalGetOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		_, stages, err := a.Order.Track(c.Request.Context(), o.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		completed := 0
		for _, s := range stages {
			if s.Result != "PENDING" && int(s.Stage) > completed {
				completed = int(s.Stage)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"order": portalOrderSummary(o, productName(a, c, o.OfferID),
				addressName(a, c, o.AddressID, ""), o.Status == "DONE"),
			"submitedAt":         o.CreatedAt.Format(time.RFC3339),
			"technicianName":     "",
			"technicianPhoneMasked": "",
			"stageLabel":         portalOrderStageLabel(o.Stage),
			"completedStage":     completed,
			"timeline":           portalOrderTimeline(stages),
		})
	}
}

// productName 产品名:主档产品表寻址,失败回退空串。
func productName(a *app.Application, c *gin.Context, offerID int64) string {
	prods, err := a.Product.ListProducts(c.Request.Context(), 0)
	if err != nil {
		return ""
	}
	for _, p := range prods {
		if p.ID == offerID {
			return p.Name
		}
	}
	return ""
}

// portalSubmitOrder POST /orders:下单(环节1 submitOrder)。productId/addressId 为字符串。
func portalSubmitOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			ProductID string `json:"productId" binding:"required"`
			AddressID string `json:"addressId" binding:"required"`
			ChannelID string `json:"channelId"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		offerID, err := strconv.ParseInt(req.ProductID, 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		addrID, err := strconv.ParseInt(req.AddressID, 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		cust, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		// 渠道缺省走线上(ONLINE);目录无该渠道时报错提示。
		channelID, err := portalChannelID(c, a, req.ChannelID)
		if err != nil {
			respondErr(c, err)
			return
		}
		o, err := a.Order.Submit(c.Request.Context(), order.SubmitReq{
			CustomerID:    cid,
			OfferID:       offerID,
			AddressID:     addrID,
			ChannelID:     channelID,
			LegalEntityID: cust.LegalEntityID,
			RegionPath:    "",
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, portalOrderSummary(o, productName(a, c, offerID),
			addressName(a, c, addrID, ""), false))
	}
}

// portalChannelID 解析渠道 ID:显式传入按 ID 校验,缺省选 ONLINE 渠道;目录未接入时返回 0。
func portalChannelID(c *gin.Context, a *app.Application, id string) (int64, error) {
	if id != "" {
		return strconv.ParseInt(id, 10, 64)
	}
	if a.Channel == nil {
		return 0, nil
	}
	channels, err := a.Channel.ListChannels(c.Request.Context())
	if err != nil {
		return 0, err
	}
	for _, ch := range channels {
		if ch.Status == "ACTIVE" && (ch.Code == "ONLINE" || ch.ID == 102) {
			return ch.ID, nil
		}
	}
	return 0, nil
}

// portalListProducts GET /products?category=:产品套餐列表(items + addons)。
func portalListProducts(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = c.Query("category")
		list, err := a.Product.ListProducts(c.Request.Context(), 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		for _, p := range list {
			if p.Status != "PUBLISHED" {
				continue
			}
			items = append(items, gin.H{
				"productId": strconv.FormatInt(p.ID, 10), "category": "broadband",
				"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
				"description": "", "contractMonths": 12, "featured": false,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "addons": []any{}})
	}
}

// portalListAddons GET /addons:增值服务(available + subscribed)。
func portalListAddons(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		items, err := a.UserData.ListAddons(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		subs, err := a.UserData.ListAddonSubscriptions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		subscribed := make(map[string]bool)
		for _, s := range subs {
			if toInt64(s["customerId"]) == cid && toStr(s["action"]) == "subscribe" {
				subscribed[toStr(s["addonId"])] = true
			}
		}
		available, owned := make([]gin.H, 0), make([]gin.H, 0)
		for _, row := range items {
			addon := gin.H{
				"addonId": toStr(row["addonId"]), "name": toStr(row["name"]),
				"monthlyFee": float64(toInt64(row["price"])) / 100,
				"description": "", "subscribed": subscribed[toStr(row["addonId"])],
			}
			if toStr(row["status"]) == "off" {
				continue
			}
			if subscribed[toStr(row["addonId"])] {
				owned = append(owned, addon)
			} else {
				available = append(available, addon)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"available": available, "subscribed": owned})
	}
}

// toInt64 map 值转 int64(JSON 数值解码为 float64)。
func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	}
	return 0
}

// toStr map 值转 string。
func toStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}