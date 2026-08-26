package userapi

// 用户端 Stripe handler 实现(stripe.go 仅留路由表 + 通用辅助)。

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/stripe"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalStripeCheckout POST /payments/stripe/checkout {billNo,amount,successUrl,cancelUrl}:
// 校验账单归属 → 建托管收银台会话 → 返回跳转 url(免客户端 SDK;落账同样等回调)。
func portalStripeCheckout(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo     string  `json:"billNo" binding:"required"`
			Amount     float64 `json:"amount" binding:"required,gt=0"`
			SuccessURL string  `json:"successUrl" binding:"required,url"`
			CancelURL  string  `json:"cancelUrl" binding:"required,url"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		gw, payNo, ok := portalStripeAcquire(c, a, cid, req.BillNo)
		if !ok {
			return
		}
		meta := map[string]string{"bill_no": req.BillNo, "customer_id": strconv.FormatInt(cid, 10)}
		co, err := gw.CreateCheckout(c.Request.Context(), payNo, toCents(req.Amount), meta, req.SuccessURL, req.CancelURL)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "billNo": req.BillNo, "amount": req.Amount,
			"checkoutUrl": co.URL, "sessionId": co.SessionID,
		})
	}
}

// portalStripeIntent POST /payments/stripe/intent {billNo,amount}:
// 校验账单归属 → 建 PaymentIntent(幂等键=payNo)→ 返回 clientSecret 交前端拉起收银台。
func portalStripeIntent(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo string  `json:"billNo" binding:"required"`
			Amount float64 `json:"amount" binding:"required,gt=0"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		gw, payNo, ok := portalStripeAcquire(c, a, cid, req.BillNo)
		if !ok {
			return
		}
		meta := map[string]string{"bill_no": req.BillNo, "customer_id": strconv.FormatInt(cid, 10)}
		intent, err := gw.CreateIntent(c.Request.Context(), payNo, toCents(req.Amount), meta)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "billNo": req.BillNo, "amount": req.Amount,
			"clientSecret": intent.ClientSecret, "intentId": intent.IntentID, "currency": intent.Currency,
		})
	}
}

// portalStripeAcquire 通道就绪 + 派单 payNo(checkout/intent 共用前置);失败已回写响应。
func portalStripeAcquire(c *gin.Context, a *app.Application, cid int64, billNo string) (billing.PaymentGateway, string, bool) {
	if a.Stripe == nil || !a.Stripe.Configured(c.Request.Context()) {
		respond(c, apitypes.CodeInvalidParam, nil) // 通道未配置(无密钥/未启用)
		return nil, "", false
	}
	gw := a.PayGateway.Get("stripe")
	if gw == nil {
		respond(c, apitypes.CodeInvalidParam, nil) // 兜底:动态网关缺席
		return nil, "", false
	}
	payNo, ok := stripeAcquirePayNo(a, c, cid, billNo)
	if !ok {
		return nil, "", false
	}
	return gw, payNo, true
}

// stripeAcquirePayNo 账单归属校验 + NextNo 派单(checkout/intent 共用前置)。
func stripeAcquirePayNo(a *app.Application, c *gin.Context, cid int64, billNo string) (string, bool) {
	b, ok := findBillByNo(a, c, cid, billNo)
	if !ok {
		respond(c, apitypes.CodeNotFound, nil)
		return "", false
	}
	if b.Status == "PAID" {
		respond(c, apitypes.CodeConflict, nil)
		return "", false
	}
	payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
	if err != nil {
		respondErr(c, err)
		return "", false
	}
	return payNo, true
}

// stripeWebhook POST /webhooks/stripe:验签 → 解析 → succeeded 落账 / failed 留痕。
// 已存在同 payNo 流水直接 200(渠道重投幂等);失败返回 4xx 让 Stripe 重试。
func stripeWebhook(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := ""
		if a.Stripe != nil {
			secret = a.Stripe.WebhookSecret(c.Request.Context())
		}
		if secret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "stripe webhook not configured"})
			return
		}
		wh := stripe.Webhook{Secret: secret}
		payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
			return
		}
		if err := wh.Verify(payload, c.GetHeader("Stripe-Signature")); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ev, err := stripe.ParseEvent(payload)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if ev.PayNo == "" {
			c.JSON(http.StatusOK, gin.H{"received": true}) // 非本系统发起的意图,忽略
			return
		}
		if err := stripeSettle(a, c, ev); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"received": true})
	}
}

// stripeSettle 按事件类型落账:
// succeeded → 账单寻址成功走 RecordPayment(流水+账单 PAID 同事务)+ 自动复机;
//
//	无账单/寻址失败走 RecordTopup(流水+余额同事务,充值到账);
//
// failed → 落 FAILED 流水留痕(无余额动作);重投(同 payNo 已落账/并发唯一冲突)幂等通过。
func stripeSettle(a *app.Application, c *gin.Context, ev stripe.Event) error {
	if ev.Type != "payment_intent.succeeded" && ev.Type != "payment_intent.payment_failed" {
		return nil
	}
	// 渠道重投幂等:同 payNo 已落账 → 直接 200。
	if paid, err := a.Billing.PaymentExistsByPayNo(c.Request.Context(), ev.PayNo); err != nil {
		return err
	} else if paid {
		return nil
	}
	amount := float64(ev.AmountCents) / 100
	if ev.Type == "payment_intent.succeeded" {
		if ev.BillNo != "" && ev.CustomerID > 0 {
			if b, ok := findBillByNo(a, c, ev.CustomerID, ev.BillNo); ok {
				_, err := a.Billing.RecordPayment(c.Request.Context(), billing.Payment{
					PayNo: ev.PayNo, BillID: b.BillID, CustomerID: ev.CustomerID,
					Amount: amount, Method: "card", Status: "SUCCESS",
				})
				if err != nil {
					return err
				}
				a.ResumeAfterPayment(c.Request.Context(), ev.CustomerID)
				return nil
			}
		}
		// 充值意图 / 账单寻址失败:充值落账(流水+余额同事务)。
		if ev.CustomerID == 0 {
			return nil // 无归属线索不落账(防孤儿),ack 等渠道重试
		}
		_, err := a.Billing.RecordTopup(c.Request.Context(), billing.Payment{
			PayNo: ev.PayNo, CustomerID: ev.CustomerID, Amount: amount, Method: "card", Status: "SUCCESS",
		})
		return settleErr(err)
	}
	// 失败留痕:无余额动作,仅落 FAILED 流水(customer_id 归属;无归属不落账)。
	if ev.CustomerID == 0 {
		return nil
	}
	_, err := a.Billing.CreatePayment(c.Request.Context(), billing.Payment{
		PayNo: ev.PayNo, CustomerID: ev.CustomerID, Amount: amount, Method: "card", Status: "FAILED",
	})
	return err
}

// settleErr 落账错误:并发唯一冲突(payNo 已落账)按幂等成功,其余上抛让渠道重试。
func settleErr(err error) error {
	if err == nil || errors.Is(err, billing.ErrPayNoExists) {
		return nil
	}
	return err
}
