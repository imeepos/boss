package userapi

// Stripe 卡收单:发起收款意图 + 渠道回调落账。
// 记账事实源在 payments/账单状态机(裁定 2026-08-20):发起只建意图不落账,
// webhook 验签后按 metadata 寻账单,RecordPayment 同事务落流水并置 PAID;pay_no 唯一幂等。

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
)

// registerStripeRoutes pub:回调(渠道服务器调用,无客户 JWT);uauth:发起收款。
func registerStripeRoutes(pub, uauth *gin.RouterGroup, a *app.Application) {
	uauth.POST("/payments/stripe/intent", portalStripeIntent(a))
	uauth.POST("/payments/stripe/checkout", portalStripeCheckout(a))
	pub.POST("/webhooks/stripe", stripeWebhook(a))
	registerStripePayDone(pub)
}

// registerStripePayDone 收银台回跳页(/pay/stripe/{done,cancel}):落账等 webhook 异步完成,
// 静态提示回 App 查缴费记录,无业务逻辑。路由登记走契约门禁(check-contract-sync A)。
func registerStripePayDone(pub *gin.RouterGroup) {
	for _, p := range []string{"/pay/stripe/done", "/pay/stripe/cancel"} {
		pub.GET(p, stripePayResultPage)
	}
}

// stripePayResultPage 收银台回跳静态页(handler)。
func stripePayResultPage(c *gin.Context) {
	const html = `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">
<title>支付结果</title><p style="font:15px/1.8 -apple-system,sans-serif;padding:40px 20px;text-align:center">
支付处理中,请回到 App 在「缴费记录」查看结果。</p>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// toCents 元 → 分(四舍五入防浮点尾差)。
func toCents(v float64) int64 { return int64(math.Round(v * 100)) }

// findBillByNo 在客户账单里找 billNo(与 portalCreatePayment 同一归属校验口径)。
func findBillByNo(a *app.Application, c *gin.Context, cid int64, billNo string) (billing.Bill, bool) {
	bills, err := a.Billing.ListBills(c.Request.Context(), cid)
	if err != nil {
		respondErr(c, err)
		return billing.Bill{}, false
	}
	for _, b := range bills {
		if b.BillNo == billNo {
			return b, true
		}
	}
	return billing.Bill{}, false
}