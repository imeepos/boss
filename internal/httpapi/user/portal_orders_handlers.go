package userapi

// 用户端门户 Order 域 handler 实现。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalListOrders GET /orders?status=&page=&pageSize=:我的订单列表
// (status: all/in_progress/done/cancelled;分页 page 从 1 起,pageSize 默认 10,多取 1 条判 hasMore)。
func portalListOrders(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		page, pageSize := portalListOrdersPaging(c)
		list, err := a.Order.List(c.Request.Context(), order.OrderQuery{
			Status: statusFilter(c.DefaultQuery("status", "all")), CustomerID: cid,
			Limit: pageSize + 1, Offset: (page - 1) * pageSize,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		hasMore, items := portalOrderPageItems(a, c, list, pageSize)
		respond(c, apitypes.CodeOK, gin.H{
			"items": items, "page": page, "pageSize": pageSize, "hasMore": hasMore,
		})
	}
}

// portalListOrdersPaging 解析分页参数(limit 上限 50)。
func portalListOrdersPaging(c *gin.Context) (int, int) {
	page := portalPositiveQuery(c, "page", 1)
	pageSize := portalPositiveQuery(c, "pageSize", 10)
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

// portalOrderPageItems 列表读模型 → 契约 items + hasMore(多取 1 条判末页)。
func portalOrderPageItems(a *app.Application, c *gin.Context, list []order.OrderListItem, pageSize int) (bool, []gin.H) {
	hasMore := len(list) > pageSize
	if hasMore {
		list = list[:pageSize]
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
	return hasMore, items
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
		respond(c, apitypes.CodeOK, portalGetOrderPayload(a, c, cid, o, stages))
	}
}

// portalGetOrderPayload 订单详情视图聚合。
func portalGetOrderPayload(a *app.Application, c *gin.Context, cid int64, o *order.Order, stages []order.StageLog) gin.H {
	techName, techPhone, hasTech := portalTechnician(a, c, cid, o.ID)
	if !hasTech {
		techName, techPhone = "", ""
	}
	return gin.H{
		"order": portalOrderSummary(o, productName(a, c, o.OfferID),
			portalOrderAddress(a, c, cid, o), o.Status == "DONE"),
		"submitedAt":            o.CreatedAt.Format(time.RFC3339),
		"technicianName":        techName,
		"technicianPhoneMasked": portalMaskPhone(techPhone),
		"stageLabel":            portalOrderStageLabel(o.Stage),
		"completedStage":        portalCompletedStage(stages),
		"timeline":              portalOrderTimeline(stages),
	}
}

// portalOrderTechnicianContact GET /orders/:orderNo/technician-contact:
// 订单装维师傅明文联系方式(派单工单回填,归属校验)。
func portalOrderTechnicianContact(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		name, phone, ok := portalTechnician(a, c, cid, o.ID)
		if !ok {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "name": name, "phone": phone})
	}
}

// portalSubmitOrder POST /orders:下单(环节1 submitOrder)。productId/addressId 为字符串。
func portalSubmitOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			ProductID   string `json:"productId" binding:"required"`
			AddressID   string `json:"addressId" binding:"required"`
			ChannelID   string `json:"channelId"`
			BillingMode string `json:"billingMode"` // PREPAID/POSTPAID,空回退 POSTPAID
			BuyMonths   int    `json:"buyMonths"`   // 预缴月数,0=按月缴;预付费环节4 按 N 月收款
			RequestID   string `json:"requestId"`   // 可选幂等键(000116):重放返回已有订单
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		offerID, addrID, channelID, ok := portalSubmitOrderParse(c, a, req.ProductID, req.AddressID, req.ChannelID)
		if !ok {
			return
		}
		o, err := a.Order.Submit(c.Request.Context(), order.SubmitReq{
			CustomerID:  cid,
			OfferID:     offerID,
			AddressID:   addrID,
			ChannelID:   channelID,       // 归属由安装地址服务端推导(2026-08-20 裁定)
			BillingMode: req.BillingMode, // 付费模式客户选定(2026-08-22 裁定)
			BuyMonths:   req.BuyMonths,   // 预缴月数(000104)
			RequestID:   req.RequestID,   // 幂等键(000116)
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, portalOrderSummary(o, productName(a, c, offerID),
			addressName(a, c, addrID, ""), false))
	}
}

// portalSubmitOrderParse 解析请求参数为强类型 ID(解析失败或渠道空 → false 已回写响应)。
func portalSubmitOrderParse(c *gin.Context, a *app.Application, productID, addressID, channelID string) (int64, int64, int64, bool) {
	offerID := portalID(productID)
	addrID := portalID(addressID)
	if offerID == 0 || addrID == 0 {
		respond(c, apitypes.CodeInvalidParam, nil)
		return 0, 0, 0, false
	}
	chID, err := portalChannelID(c, a, channelID)
	if err != nil {
		respondErr(c, err)
		return 0, 0, 0, false
	}
	if chID == 0 {
		respond(c, apitypes.CodeInvalidParam, nil)
		return 0, 0, 0, false
	}
	return offerID, addrID, chID, true
}
