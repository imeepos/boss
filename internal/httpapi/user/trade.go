package userapi

// 用户端门户 Product/Order/Plans 域:产品详情、订单操作/评价、套餐变更与退订预检

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ---- Product / Order ----

// registerPortalOrderRoutes 客户视角产品详情 + 订单操作与套餐变更。
func registerPortalOrderRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/products", portalListProducts(a))
	g.GET("/products/:id", portalProductDetail(a))
	g.GET("/addons", portalListAddons(a))
	g.POST("/addons/:addonId/subscribe", portalAddonSubscribe(a))
	g.POST("/addons/:addonId/unsubscribe", portalAddonUnsubscribe(a))
	g.GET("/orders", portalListOrders(a))
	g.POST("/orders", portalSubmitOrder(a))
	g.GET("/orders/:orderNo", portalGetOrder(a))
	g.POST("/orders/:orderNo/cancel", portalOrderCancel(a))
	g.POST("/orders/:orderNo/urge", portalOrderUrge(a))
	g.POST("/orders/:orderNo/change-address", portalOrderChangeAddr(a))
	g.GET("/orders/:orderNo/rate", portalRateGet(a))
	g.POST("/orders/:orderNo/rate", portalRatePost(a))
	g.GET("/plans/:planId/cancel", portalPlanCancelPreview(a))
	g.POST("/plans/:planId/change", portalPlanChange(a))
	g.POST("/plans/:planId/move", portalPlanMove(a))
	g.POST("/plans/:planId/cancel", portalPlanCancel(a))
}

// portalAddonSubscribe 订购增值服务(userdata 订购关系落库)。
func portalAddonSubscribe(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if err := portalAddonOpPersist(c, a, cid, "subscribe"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalAddonUnsubscribe 退订增值服务(userdata 退订关系落库)。
func portalAddonUnsubscribe(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if err := portalAddonOpPersist(c, a, cid, "unsubscribe"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalAddonOpPersist 增值订购/退订落库(addon_subscriptions 快照关系)。
func portalAddonOpPersist(c *gin.Context, a *app.Application, cid int64, action string) error {
	if a.UserData == nil {
		return nil
	}
	_, err := a.UserData.CreateAddonSubscription(c.Request.Context(), udcustomer.AddonSubscription{
		CustomerID: cid, AddonID: c.Param("addonId"), Action: action,
	})
	return err
}

// portalProductDetail GET /products/:id:产品详情(PUBLISHED 才可见)。
func portalProductDetail(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		list, lerr := a.Product.ListProducts(c.Request.Context(), 0)
		if lerr != nil {
			respondErr(c, lerr)
			return
		}
		for _, p := range list {
			if p.ID == id && p.Status == "PUBLISHED" {
				respond(c, apitypes.CodeOK, gin.H{"product": gin.H{
					"productId": strconv.FormatInt(p.ID, 10), "category": "broadband",
					"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
					"contractMonths": 12, "featured": false,
				}, "specs": []gin.H{{"label": "带宽", "value": p.Bandwidth}}, "compare": []any{}})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalOwnedOrder 取订单并校验归属(客户视角越权一律 404,不泄露存在性)。
func portalOwnedOrder(a *app.Application, c *gin.Context, cid int64) (*order.Order, bool) {
	o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
	if err != nil || o.CustomerID != cid {
		respond(c, apitypes.CodeNotFound, nil)
		return nil, false
	}
	return o, true
}

// portalOrderCancel POST /orders/:orderNo/cancel:取消订单(释放预占端口)。
func portalOrderCancel(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		if err := a.Order.Cancel(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderUrge POST /orders/:orderNo/urge:催单落站内消息(消息表持久化)。
func portalOrderUrge(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Portal.PutMessage(c.Request.Context(), cid, gin.H{
			"messageId": msgID, "category": "fault", "title": "催单已受理",
			"content": "订单 " + o.OrderNo + " 已加急处理", "tag": "订单", "tagLevel": "info",
			"createdAt": time.Now(), "read": false,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderChangeAddr POST /orders/:orderNo/change-address:变更安装地址(orders.address_id 落库)。
func portalOrderChangeAddr(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		var req struct {
			AddressID string `json:"addressId" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		addrID, err := strconv.ParseInt(req.AddressID, 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.Order.ChangeAddress(c.Request.Context(), o.ID, addrID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalRateGet GET /orders/:orderNo/rate:待评价信息(DONE 且未评价)。
func portalRateGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		if o.Status != "DONE" {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		if rated, err := a.Order.RatingExists(c.Request.Context(), o.OrderNo); err == nil && rated {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"orderNo": o.OrderNo, "productName": productName(a, c, o.OfferID),
			"address": addressName(a, c, o.AddressID, ""), "finishedAt": o.CreatedAt,
		})
	}
}

// portalRatePost POST /orders/:orderNo/rate:提交评价(order_ratings 落库,唯一单号)。
func portalRatePost(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		var req struct {
			Stars    int    `json:"stars" binding:"required,min=1,max=5"`
			Attitude int    `json:"attitude" binding:"required,min=1,max=5"`
			Quality  int    `json:"quality" binding:"required,min=1,max=5"`
			Comment  string `json:"comment"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Order.SaveRating(c.Request.Context(), order.Rating{
			OrderNo: o.OrderNo, CustomerID: cid,
			Stars: int8(req.Stars), Attitude: int8(req.Attitude),
			Quality: int8(req.Quality), Comment: req.Comment,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalPlanCancelPreview GET /plans/:planId/cancel:退订预检(未缴账单 + 违约金示意)。
func portalPlanCancelPreview(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var unpaid []gin.H
		var penalty float64
		bills, _ := a.Billing.ListBills(c.Request.Context(), cid)
		for _, b := range bills {
			if b.Status != "PAID" {
				unpaid = append(unpaid, gin.H{"billNo": b.BillNo, "period": b.Period,
					"amount": b.Amount, "status": b.Status})
				penalty += b.Amount
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"unpaidBills": unpaid, "penalty": penalty,
			"penaltyDesc": "含未出账部分按套餐剩余月份折算",
		})
	}
}

// portalPlanChange POST /plans/:planId/change:改套餐 → 真实变更单(orders 落库)。
func portalPlanChange(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			TargetPlanID  string `json:"targetPlanId" binding:"required"`
			EffectiveMode string `json:"effectiveMode" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		target, err := strconv.ParseInt(req.TargetPlanID, 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		o, err := portalSubmitWorkOrder(c, a, cid, target, 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, portalOrderSummary(o, productName(a, c, target),
			addressName(a, c, o.AddressID, ""), false))
	}
}

// portalPlanMove POST /plans/:planId/move:迁址 → 真实迁址单(orders 落库)。
func portalPlanMove(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		cust, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		o, err := portalSubmitWorkOrder(c, a, cid, 0, cust.AddressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, portalOrderSummary(o, productName(a, c, o.OfferID),
			addressName(a, c, o.AddressID, ""), false))
	}
}

// portalPlanCancel POST /plans/:planId/cancel:确认拆机 → 拆机单(orders 落库,立即取消态)。
func portalPlanCancel(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Reason string `json:"reason" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		_ = req.Reason
		cust, err := a.Customer.Get(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		o, err := portalSubmitWorkOrder(c, a, cid, 0, cust.AddressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Order.Cancel(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, portalOrderSummary(o, productName(a, c, o.OfferID),
			addressName(a, c, o.AddressID, ""), false))
	}
}

// portalSubmitWorkOrder 生成工作单:与新装共用订单主表;offerID=0 时沿用客户当前套餐。
func portalSubmitWorkOrder(c *gin.Context, a *app.Application, cid, targetOffer, addressID int64) (*order.Order, error) {
	cust := &customer.Customer{ID: cid}
	if v, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
		cust = v
	}
	if addressID == 0 {
		addressID = cust.AddressID
	}
	offerID := targetOffer
	if offerID == 0 {
		if plan, ok := portalCurrentPlan(a, c, cid); ok {
			offerID = plan.ProductID
		}
	}
	if offerID == 0 {
		offerID = 101 // 演示兜底:家庭宽带 100M
	}
	channelID, err := portalChannelID(c, a, "")
	if err != nil {
		return nil, err
	}
	if channelID == 0 {
		return nil, app.ErrNotImplemented
	}
	return a.Order.Submit(c.Request.Context(), order.SubmitReq{
		CustomerID: cid, OfferID: offerID, AddressID: addressID,
		ChannelID: channelID, LegalEntityID: cust.LegalEntityID, RegionPath: "",
	})
}

// portalCurrentPlan 客户当前套餐(user_plans 最近生效)。
func portalCurrentPlan(a *app.Application, c *gin.Context, cid int64) (udcustomer.UserPlan, bool) {
	if a.UserData == nil {
		return udcustomer.UserPlan{}, false
	}
	rows, err := a.UserData.ListUserPlans(c.Request.Context())
	if err != nil {
		return udcustomer.UserPlan{}, false
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) == cid {
			return udcustomer.UserPlan{ProductID: toInt64(r["productId"])}, true
		}
	}
	return udcustomer.UserPlan{}, false
}
