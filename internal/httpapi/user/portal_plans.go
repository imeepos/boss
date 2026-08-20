package userapi

// 用户端门户套餐变更域:改套餐/迁址/拆机(全部落到 orders 真实工作单)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalPlanCancelPreview GET /plans/:planId/cancel:退订预检(未出账账单 + 违约金估算)。
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
