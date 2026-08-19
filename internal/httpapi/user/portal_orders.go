package userapi

// 用户端门户 Order 域核心端点:订单列表/下单/详情/时间轴 + 渠道解析。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

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
		techName, techPhone, hasTech := portalTechnician(a, c, cid, o.ID)
		if !hasTech {
			techName, techPhone = "", ""
		}
		respond(c, apitypes.CodeOK, gin.H{
			"order": portalOrderSummary(o, productName(a, c, o.OfferID),
				addressName(a, c, o.AddressID, ""), o.Status == "DONE"),
			"submitedAt":            o.CreatedAt.Format(time.RFC3339),
			"technicianName":        techName,
			"technicianPhoneMasked": portalMaskPhone(techPhone),
			"stageLabel":            portalOrderStageLabel(o.Stage),
			"completedStage":        completed,
			"timeline":              portalOrderTimeline(stages),
		})
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
			ProductID string `json:"productId" binding:"required"`
			AddressID string `json:"addressId" binding:"required"`
			ChannelID string `json:"channelId"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		offerID := portalID(req.ProductID)
		addrID := portalID(req.AddressID)
		if offerID == 0 || addrID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		cust, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		channelID, err := portalChannelID(c, a, req.ChannelID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if channelID == 0 {
			respond(c, apitypes.CodeInvalidParam, nil)
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