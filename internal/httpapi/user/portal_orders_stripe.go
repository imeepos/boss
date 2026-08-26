package userapi

// 用户端门户 Order 域 → Stripe 卡收单桥接:
// 客户端提交订单后(POST /orders),客户端调本端点为该订单的"订单首期"建账 + 发起收款,
// 落账走既有 /webhooks/stripe 链路。客户归属校验复用 portalOwnedOrder,账单建账复用 billing.CreateBill。

import (
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalOrderStripeIntent POST /orders/:orderNo/stripe-intent:
// 校验订单归属 → 按订单当前价(产品月费 × buyMonths,buyMonths<=0 视作 1)建账(同订单幂等)→
// 通道就绪 + 派单 payNo → CreateIntent(幂等键 payNo)→ 回 clientSecret 交前端 PaymentSheet。
func portalOrderStripeIntent(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		var req struct {
			BuyMonths int `json:"buyMonths"`
		}
		_ = c.ShouldBindJSON(&req) // body 可选;未传回退订单 buyMonths
		b, ok := portalOrderBillBuild(c, a, cid, o, req.BuyMonths)
		if !ok {
			return
		}
		gw, payNo, ok := portalStripeAcquire(c, a, cid, b.BillNo)
		if !ok {
			return
		}
		meta := map[string]string{
			"bill_no": b.BillNo, "order_no": o.OrderNo,
			"customer_id": strconv.FormatInt(cid, 10),
		}
		intent, err := gw.CreateIntent(c.Request.Context(), payNo, toCents(b.Amount), meta)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "billNo": b.BillNo, "orderNo": o.OrderNo,
			"amount": b.Amount, "clientSecret": intent.ClientSecret,
			"intentId": intent.IntentID, "currency": intent.Currency,
		})
	}
}

// portalOrderStripeCheckout POST /orders/:orderNo/stripe-checkout:
// 与 intent 同前置(校验/建账),差异在 CreateCheckout(免客户端 SDK;前端跳 checkoutUrl)。
func portalOrderStripeCheckout(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		var req struct {
			BuyMonths  int    `json:"buyMonths"`
			SuccessURL string `json:"successUrl"`
			CancelURL  string `json:"cancelUrl"`
		}
		_ = c.ShouldBindJSON(&req)
		b, ok := portalOrderBillBuild(c, a, cid, o, req.BuyMonths)
		if !ok {
			return
		}
		gw, payNo, ok := portalStripeAcquire(c, a, cid, b.BillNo)
		if !ok {
			return
		}
		if req.SuccessURL == "" {
			req.SuccessURL = defaultStripeSuccessURL()
		}
		if req.CancelURL == "" {
			req.CancelURL = defaultStripeCancelURL()
		}
		if !isHTTPLikeURL(req.SuccessURL) || !isHTTPLikeURL(req.CancelURL) {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		meta := map[string]string{
			"bill_no": b.BillNo, "order_no": o.OrderNo,
			"customer_id": strconv.FormatInt(cid, 10),
		}
		co, err := gw.CreateCheckout(c.Request.Context(), payNo, toCents(b.Amount), meta,
			req.SuccessURL, req.CancelURL)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "billNo": b.BillNo, "orderNo": o.OrderNo,
			"amount": b.Amount, "checkoutUrl": co.URL, "sessionId": co.SessionID,
		})
	}
}

// portalOrderBillBuild 按订单建账:同订单同 period 复用已建账单(避免重复);金额=月费 × 月数(0 视作 1)。
// 失败已回写响应,返回 (bill, true) 继续后续,或 (zero, false) 终止。
func portalOrderBillBuild(c *gin.Context, a *app.Application, cid int64, o *order.Order, buyMonths int) (billing.Bill, bool) {
	if a.Billing == nil {
		respond(c, apitypes.CodeInternal, nil)
		return billing.Bill{}, false
	}
	if o.Status == "DONE" || o.Status == "CANCELLED" {
		respond(c, apitypes.CodeConflict, nil)
		return billing.Bill{}, false
	}
	months := buyMonths
	if months <= 0 {
		months = o.BuyMonths
	}
	if months <= 0 {
		months = 1
	}
	amount, ok := portalOrderAmountResolve(c, a, o)
	if !ok {
		return billing.Bill{}, false
	}
	amount *= float64(months)
	period := portalOrderBillPeriod(o)
	if existing, ok := portalOrderFindBill(a, c, cid, o, period); ok && existing.Status != "PAID" {
		return existing, true
	}
	cu, err := a.Customer.Get(c.Request.Context(), cid)
	if err != nil {
		respondErr(c, err)
		return billing.Bill{}, false
	}
	billNo, err := a.Portal.NextNo(c.Request.Context(), "BILL")
	if err != nil {
		respondErr(c, err)
		return billing.Bill{}, false
	}
	billID, err := a.Billing.CreateBill(c.Request.Context(), billing.Bill{
		BillNo: billNo, CustomerID: cid, CustomerName: cu.Name,
		LegalEntityID: o.LegalEntityID, LegalEntityName: "",
		RegionID: cu.RegionID, RegionName: cu.RegionName,
		Period: period, Amount: amount, Status: "UNPAID",
	})
	if err != nil && !errors.Is(err, billing.ErrForeignKeyViolation) {
		respondErr(c, err)
		return billing.Bill{}, false
	}
	return billing.Bill{
		BillID: billID, BillNo: billNo, CustomerID: cid, CustomerName: cu.Name,
		LegalEntityID: o.LegalEntityID, Period: period, Amount: amount, Status: "UNPAID",
	}, true
}

// portalOrderFindBill 同订单同 period 找已有账单(UNPAID 复用);UNIQUE(customer_id, period) 已保证唯一。
func portalOrderFindBill(a *app.Application, c *gin.Context, cid int64, o *order.Order, period string) (billing.Bill, bool) {
	bills, err := a.Billing.ListBills(c.Request.Context(), cid)
	if err != nil {
		return billing.Bill{}, false
	}
	for _, b := range bills {
		if b.Period == period && b.CustomerID == cid {
			return b, true
		}
	}
	return billing.Bill{}, false
}

// portalOrderAmount 订单对应月费:复用 ProductService 拿产品目录(月费权威源)。
// 产品下架/不存在时返回 ErrOfferUnavailable(由调用方映射为 40400,文案"产品已下架")。
func portalOrderAmount(a *app.Application, c *gin.Context, o *order.Order) (float64, error) {
	if a.Product == nil {
		return 0, errors.New("order stripe: product service not wired")
	}
	offers, err := a.Product.ListProducts(c.Request.Context(), 0)
	if err != nil {
		return 0, err
	}
	for _, p := range offers {
		if p.ID == o.OfferID {
			return p.MonthlyFee, nil
		}
	}
	return 0, order.ErrOfferNotOrderable
}

// portalOrderAmountResolve 取订单月费,产品不可用时回写 40400 响应并返回 (0, false)。
func portalOrderAmountResolve(c *gin.Context, a *app.Application, o *order.Order) (float64, bool) {
	amount, err := portalOrderAmount(a, c, o)
	if errors.Is(err, order.ErrOfferNotOrderable) {
		respond(c, apitypes.CodeNotFound, nil)
		return 0, false
	}
	if err != nil {
		respondErr(c, err)
		return 0, false
	}
	return amount, true
}

// portalOrderBillPeriod 账单账期:订单创建月(YYYY-MM)。同月内复用同账期(UNIQUE 约束)。
func portalOrderBillPeriod(o *order.Order) string {
	return o.CreatedAt.In(time.Local).Format("2006-01")
}

// defaultStripeSuccessURL/Cancel 收银台回跳静态页(走 registerStripePayDone 注册)。
// 此处取相对路径即可,前端可经 Api.base 拼绝对值。
func defaultStripeSuccessURL() string { return "/pay/stripe/done" }
func defaultStripeCancelURL() string  { return "/pay/stripe/cancel" }

// isHTTPLikeURL 简易 URL 校验:解析成功 + scheme 为 http/https;相对路径(/pay/...)也算合法。
// 比 strict URL 校验宽松,符合 Stripe 收银台回跳允许相对路径的实战用法。
func isHTTPLikeURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u == nil || u.String() == "" {
		return false
	}
	if u.Scheme == "" {
		return u.Path != "" // 相对路径,有 path 即合法
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
