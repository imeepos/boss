package userapi

// Stripe 收单端点单测:发起意图(归属校验+通道未配置)与 webhook(验签/落账/幂等)。

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/stripe"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeStripeGW 桩网关:记录入参,返回固定意图。
type fakeStripeGW struct {
	lastPayNo string
	lastCents int64
	lastMeta  map[string]string
}

func (f *fakeStripeGW) Channel() string { return "stripe" }
func (f *fakeStripeGW) CreateIntent(_ context.Context, payNo string, cents int64, meta map[string]string) (billing.PayIntent, error) {
	f.lastPayNo, f.lastCents, f.lastMeta = payNo, cents, meta
	return billing.PayIntent{IntentID: "pi_t", ClientSecret: "pi_t_secret", AmountCents: cents, Currency: "php"}, nil
}
func (f *fakeStripeGW) CreateCheckout(_ context.Context, payNo string, cents int64, meta map[string]string,
	_, _ string) (billing.PayCheckout, error) {
	f.lastPayNo, f.lastCents, f.lastMeta = payNo, cents, meta
	return billing.PayCheckout{SessionID: "cs_t", URL: "https://checkout.example/cs_t"}, nil
}

// settleBilling 桩 Billing:记录 RecordPayment/CreatePayment 落账入参。
type settleBilling struct {
	bills    []billing.Bill
	pays     []billing.Payment
	recorded []billing.Payment
	created  []billing.Payment
}

func (f *settleBilling) ListBills(context.Context, int64) ([]billing.Bill, error) {
	return f.bills, nil
}
func (f *settleBilling) CreateBill(_ context.Context, b billing.Bill) (int64, error) {
	if b.BillID == 0 {
		b.BillID = int64(len(f.bills) + 1)
	}
	f.bills = append(f.bills, b)
	return b.BillID, nil
}
func (f *settleBilling) GetBill(context.Context, int64) (*billing.Bill, error) { return nil, nil }
func (f *settleBilling) ListPayments(context.Context, int64) ([]billing.Payment, error) {
	return f.pays, nil
}
func (f *settleBilling) ListPaymentsByCustomer(context.Context, int64) ([]billing.Payment, error) {
	return f.pays, nil
}
func (f *settleBilling) CreatePayment(_ context.Context, p billing.Payment) (int64, error) {
	f.created = append(f.created, p)
	f.pays = append(f.pays, p)
	return int64(len(f.pays)), nil
}
func (f *settleBilling) PaymentExistsByPayNo(_ context.Context, payNo string) (bool, error) {
	for _, p := range f.pays {
		if p.PayNo == payNo {
			return true, nil
		}
	}
	return false, nil
}
func (f *settleBilling) RecordTopup(_ context.Context, p billing.Payment) (int64, error) {
	f.created = append(f.created, p)
	f.pays = append(f.pays, p)
	return int64(len(f.pays)), nil
}
func (f *settleBilling) RecordPayment(_ context.Context, p billing.Payment) (int64, error) {
	f.recorded = append(f.recorded, p)
	f.pays = append(f.pays, p)
	return int64(len(f.pays)), nil
}
func (f *settleBilling) RecordPaymentWithCoupon(_ context.Context, p billing.Payment) (billing.PaymentReceipt, error) {
	id, err := f.RecordPayment(context.Background(), p)
	return billing.PaymentReceipt{PaymentID: id, Amount: p.Amount}, err
}
func (f *settleBilling) GenerateBills(context.Context, string) (int, error) { return 0, nil }
func (f *settleBilling) RefundPayment(_ context.Context, _ int64, _ string) (*billing.Payment, error) {
	return nil, nil
}

// newStripeRouter gw 为 nil 表示通道未配置(无 APIKey → 动态判未配置,发起端点 400)。
func newStripeRouter(bill billing.BillingService, gw billing.PaymentGateway, wh stripe.Webhook) *gin.Engine {
	return newStripeRouterFull(bill, gw, wh, nil, nil, nil)
}

// newStripeRouterFull 支持注入订单(portalOrderStripe* 端点需要);products/order 均可为 nil。
func newStripeRouterFull(bill billing.BillingService, gw billing.PaymentGateway, wh stripe.Webhook,
	products []customer.ProductOffer, byNo *order.Order, byNoErr error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	apiKey := ""
	if gw != nil {
		apiKey = "sk_test_x"
	}
	dyn := stripe.NewDynamic(func(context.Context) (stripe.Config, error) {
		return stripe.Config{Enabled: true, APIKey: apiKey, WebhookSecret: wh.Secret, Currency: "php"}, nil
	})
	a := &app.Application{
		Customer: &userPortalCustSvc{c: userPortalCust()}, Billing: bill,
		Product: &fakeProduct{list: products},
		Order:   &fakeOrder{byNo: byNo, byNoErr: byNoErr},
		Portal:  portal.NewMemory(), Stripe: dyn,
	}
	if gw != nil {
		a.PayGateway = billing.NewPaymentGatewayRegistry(gw)
	} else {
		a.PayGateway = billing.NewPaymentGatewayRegistry()
	}
	r := gin.New()
	Register(r, a, auth.NewManager("test-secret", time.Hour))
	return r
}

func stripeCustToken(custID int64) string {
	tok, _ := signCustomerToken(auth.NewManager("test-secret", time.Hour), custID, "13800001234")
	return tok
}

func webhookDo(r *gin.Engine, payload, sig string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/user/v1/webhooks/stripe", strings.NewReader(payload))
	req.Header.Set("Stripe-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestStripeIntent 契约:意图端点校验账单归属,返回 clientSecret;未配置通道 400。
func TestStripeIntent(t *testing.T) {
	cust := userPortalCust()
	bill := &settleBilling{bills: []billing.Bill{
		{BillID: 1, BillNo: "B-1", CustomerID: cust.ID, Amount: 99, Status: "UNPAID"},
		{BillID: 2, BillNo: "B-2", CustomerID: cust.ID, Amount: 50, Status: "PAID"},
	}}
	gw := &fakeStripeGW{}
	tok := stripeCustToken(cust.ID)
	r := newStripeRouter(bill, gw, stripe.Webhook{Secret: "whsec_x"})

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/payments/stripe/intent",
		`{"billNo":"B-1","amount":99}`, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) || data["clientSecret"] != "pi_t_secret" || data["payNo"] == nil {
		t.Fatalf("intent resp=%s", w.Body.String())
	}
	if gw.lastCents != 9900 || gw.lastMeta["bill_no"] != "B-1" {
		t.Fatalf("gw called with cents=%d meta=%v", gw.lastCents, gw.lastMeta)
	}

	// 已缴账单 → 409;未配置网关 → 400;他人账单查不到 → 404。
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/payments/stripe/intent", `{"billNo":"B-2","amount":50}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeConflict) {
		t.Fatalf("paid bill resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/payments/stripe/intent", `{"billNo":"B-X","amount":50}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeNotFound) {
		t.Fatalf("foreign bill resp=%s", w.Body.String())
	}
	r2 := newStripeRouter(bill, nil, stripe.Webhook{Secret: "whsec_x"})
	w = userPortalDo(r2, http.MethodPost, "/api/user/v1/payments/stripe/intent", `{"billNo":"B-1","amount":99}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeInvalidParam) {
		t.Fatalf("unconfigured gateway resp=%s", w.Body.String())
	}
}

// TestStripeCheckout 契约:托管收银台端点返回跳转 url;已缴 409。
func TestStripeCheckout(t *testing.T) {
	cust := userPortalCust()
	bill := &settleBilling{bills: []billing.Bill{
		{BillID: 1, BillNo: "B-1", CustomerID: cust.ID, Amount: 99, Status: "UNPAID"},
		{BillID: 2, BillNo: "B-2", CustomerID: cust.ID, Amount: 50, Status: "PAID"},
	}}
	gw := &fakeStripeGW{}
	tok := stripeCustToken(cust.ID)
	r := newStripeRouter(bill, gw, stripe.Webhook{Secret: "whsec_x"})
	body := `{"billNo":"B-1","amount":99,"successUrl":"https://app.example/ok","cancelUrl":"https://app.example/cancel"}`

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/payments/stripe/checkout", body, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) || data["checkoutUrl"] != "https://checkout.example/cs_t" || data["payNo"] == nil {
		t.Fatalf("checkout resp=%s", w.Body.String())
	}
	if gw.lastCents != 9900 || gw.lastMeta["bill_no"] != "B-1" {
		t.Fatalf("gw cents=%d meta=%v", gw.lastCents, gw.lastMeta)
	}
	paid := `{"billNo":"B-2","amount":50,"successUrl":"https://a.example/ok","cancelUrl":"https://a.example/cancel"}`
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/payments/stripe/checkout", paid, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeConflict) {
		t.Fatalf("paid bill resp=%s", w.Body.String())
	}
}

// TestStripePayDone 回跳页:done/cancel 返回静态 HTML(收银台回跳不 404)。
func TestStripePayDone(t *testing.T) {
	r := newStripeRouter(&settleBilling{}, nil, stripe.Webhook{Secret: "whsec_x"})
	for _, p := range []string{"/api/user/v1/pay/stripe/done", "/api/user/v1/pay/stripe/cancel"} {
		w := userPortalDo(r, http.MethodGet, p, "", "")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "缴费记录") {
			t.Fatalf("pay done %s resp=%d", p, w.Code)
		}
	}
}

// TestStripeWebhook 契约:验签通过+succeeded 经 RecordPayment 落账;篡改签名 400;重投幂等。
func TestStripeWebhook(t *testing.T) {
	cust := userPortalCust()
	bill := &settleBilling{bills: []billing.Bill{{BillID: 1, BillNo: "B-1", CustomerID: cust.ID, Amount: 99, Status: "UNPAID"}}}
	wh := stripe.Webhook{Secret: "whsec_x"}
	r := newStripeRouter(bill, nil, wh)
	payload := fmt.Sprintf(`{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_9","amount":9900,
		"currency":"php","metadata":{"pay_no":"PAY-S1","bill_no":"B-1","customer_id":"%d"}}}}`, cust.ID)

	w := webhookDo(r, payload, wh.SignPayload([]byte(payload), time.Now()))
	if w.Code != http.StatusOK {
		t.Fatalf("webhook resp=%d %s", w.Code, w.Body.String())
	}
	if len(bill.recorded) != 1 {
		t.Fatalf("recorded=%v", bill.recorded)
	}
	p := bill.recorded[0]
	if p.PayNo != "PAY-S1" || p.BillID != 1 || p.Amount != 99 || p.Method != "card" || p.Status != "SUCCESS" {
		t.Fatalf("recorded payment: %+v", p)
	}

	// 重投同事件:不再新增落账。
	w = webhookDo(r, payload, wh.SignPayload([]byte(payload), time.Now()))
	if w.Code != http.StatusOK || len(bill.recorded) != 1 || len(bill.pays) != 1 {
		t.Fatalf("replay resp=%d recorded=%d pays=%d", w.Code, len(bill.recorded), len(bill.pays))
	}

	// 篡改签名拒绝。
	w = webhookDo(r, `{"type":"payment_intent.succeeded"}`, "t=1,v1=bad")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad sig resp=%d", w.Code)
	}
}

// TestStripeWebhookTopup 无账单意图(customer_id 有、bill_no 空):落充值流水(流水+余额同事务);
// 重投幂等不重复落账。
func TestStripeWebhookTopup(t *testing.T) {
	cust := userPortalCust()
	bill := &settleBilling{}
	wh := stripe.Webhook{Secret: "whsec_x"}
	r := newStripeRouter(bill, nil, wh)
	payload := fmt.Sprintf(`{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_t","amount":5000,
		"currency":"php","metadata":{"pay_no":"PAY-T1","customer_id":"%d"}}}}`, cust.ID)

	w := webhookDo(r, payload, wh.SignPayload([]byte(payload), time.Now()))
	if w.Code != http.StatusOK || len(bill.created) != 1 || bill.created[0].CustomerID != cust.ID ||
		bill.created[0].Amount != 50 || bill.created[0].BillID != 0 {
		t.Fatalf("topup resp=%d created=%v", w.Code, bill.created)
	}
	// 重投:不再新增落账(余额不重复到账)。
	w = webhookDo(r, payload, wh.SignPayload([]byte(payload), time.Now()))
	if w.Code != http.StatusOK || len(bill.created) != 1 || len(bill.pays) != 1 {
		t.Fatalf("topup replay resp=%d created=%d pays=%d", w.Code, len(bill.created), len(bill.pays))
	}
}

// TestStripeWebhookTopupFailed 无账单失败意图:只落 FAILED 流水,不走充值落账(无余额动作)。
func TestStripeWebhookTopupFailed(t *testing.T) {
	cust := userPortalCust()
	bill := &settleBilling{}
	wh := stripe.Webhook{Secret: "whsec_x"}
	r := newStripeRouter(bill, nil, wh)
	payload := fmt.Sprintf(`{"type":"payment_intent.payment_failed","data":{"object":{"id":"pi_t","amount":5000,
		"currency":"php","metadata":{"pay_no":"PAY-TF","customer_id":"%d"}}}}`, cust.ID)

	w := webhookDo(r, payload, wh.SignPayload([]byte(payload), time.Now()))
	if w.Code != http.StatusOK || len(bill.created) != 1 || bill.created[0].Status != "FAILED" ||
		bill.created[0].CustomerID != cust.ID {
		t.Fatalf("topup failed resp=%d created=%v", w.Code, bill.created)
	}
}

// TestStripeWebhookUnattributable 无 bill_no 且无 customer_id:不落账(防孤儿),ack 等渠道重试。
func TestStripeWebhookUnattributable(t *testing.T) {
	bill := &settleBilling{}
	wh := stripe.Webhook{Secret: "whsec_x"}
	r := newStripeRouter(bill, nil, wh)
	payload := `{"type":"payment_intent.succeeded","data":{"object":{"id":"pi_t","amount":5000,
		"currency":"php","metadata":{"pay_no":"PAY-NA"}}}}`

	w := webhookDo(r, payload, wh.SignPayload([]byte(payload), time.Now()))
	if w.Code != http.StatusOK || len(bill.created) != 0 || len(bill.recorded) != 0 {
		t.Fatalf("unattributable resp=%d created=%d recorded=%d", w.Code, len(bill.created), len(bill.recorded))
	}
}

// TestOrderStripeIntent 契约:订单收款 intent 端点校验订单归属 + 自动建账 + 派 payNo + 通道就绪;
// 非归属 404;已取消/已 DONE 409;未配置通道 400;无产品月费 404。
func TestOrderStripeIntent(t *testing.T) {
	cust := userPortalCust()
	ord := &order.Order{ID: 9, OrderNo: "ORD-9", CustomerID: cust.ID, OfferID: 11,
		LegalEntityID: 1, RegionPath: "root.luzon", Status: "PENDING", BuyMonths: 0,
		CreatedAt: time.Date(2026, 8, 22, 10, 0, 0, 0, time.Local)}
	bill := &settleBilling{}
	gw := &fakeStripeGW{}
	tok := stripeCustToken(cust.ID)

	// Happy path:订单 PENDING + 产品月费 158 → 元转分 15800;同 period 复用已建账单。
	r := newStripeRouterFull(bill, gw, stripe.Webhook{Secret: "whsec_x"},
		[]customer.ProductOffer{{ID: 11, LegalEntityID: 1, Name: "300M 宽带", MonthlyFee: 158.0, Status: "PUBLISHED"}}, ord, nil)
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-intent", `{}`, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) {
		t.Fatalf("intent resp=%s", w.Body.String())
	}
	if data["billNo"] == nil || data["payNo"] == nil || data["clientSecret"] != "pi_t_secret" ||
		data["orderNo"] != "ORD-9" {
		t.Fatalf("intent data=%+v", data)
	}
	if gw.lastCents != 15800 || gw.lastMeta["bill_no"] == "" ||
		gw.lastMeta["order_no"] != "ORD-9" || gw.lastMeta["customer_id"] != "7" {
		t.Fatalf("gw cents=%d meta=%v", gw.lastCents, gw.lastMeta)
	}

	// DONE 订单 → 409;非归属订单 → 404;通道未配置 → 400;产品不存在 → 404。
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-intent", `{}`, tok)
	// 此时 bill 已建并被复用,status=UNPAID,所以再次返回 200。改测 DONE:
	r2 := newStripeRouterFull(&settleBilling{}, gw, stripe.Webhook{Secret: "whsec_x"},
		[]customer.ProductOffer{{ID: 11, MonthlyFee: 158.0, Status: "PUBLISHED"}},
		&order.Order{ID: 9, OrderNo: "ORD-9", CustomerID: cust.ID, OfferID: 11, Status: "DONE", CreatedAt: time.Now()}, nil)
	w = userPortalDo(r2, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-intent", `{}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeConflict) {
		t.Fatalf("done order resp=%s", w.Body.String())
	}
	r3 := newStripeRouterFull(&settleBilling{}, gw, stripe.Webhook{Secret: "whsec_x"}, nil,
		&order.Order{ID: 9, OrderNo: "ORD-9", CustomerID: cust.ID + 1, OfferID: 11, Status: "PENDING", CreatedAt: time.Now()}, nil)
	w = userPortalDo(r3, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-intent", `{}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeNotFound) {
		t.Fatalf("foreign order resp=%s", w.Body.String())
	}
	r4 := newStripeRouterFull(&settleBilling{}, nil, stripe.Webhook{Secret: "whsec_x"},
		[]customer.ProductOffer{{ID: 11, MonthlyFee: 158.0, Status: "PUBLISHED"}}, ord, nil)
	w = userPortalDo(r4, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-intent", `{}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeInvalidParam) {
		t.Fatalf("unconfigured gateway resp=%s", w.Body.String())
	}
	r5 := newStripeRouterFull(&settleBilling{}, gw, stripe.Webhook{Secret: "whsec_x"},
		nil, ord, nil) // products=nil → 找不到 offer → 映射 40400
	w = userPortalDo(r5, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-intent", `{}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeNotFound) {
		t.Fatalf("no offer resp=%s", w.Body.String())
	}
}

// TestOrderStripeCheckout 契约:订单 checkout 端点建账 → 返回 checkoutUrl;非法 URL 42200。
func TestOrderStripeCheckout(t *testing.T) {
	cust := userPortalCust()
	ord := &order.Order{ID: 9, OrderNo: "ORD-9", CustomerID: cust.ID, OfferID: 11,
		LegalEntityID: 1, RegionPath: "root.luzon", Status: "PENDING", BuyMonths: 0,
		CreatedAt: time.Date(2026, 8, 22, 10, 0, 0, 0, time.Local)}
	bill := &settleBilling{}
	gw := &fakeStripeGW{}
	tok := stripeCustToken(cust.ID)

	r := newStripeRouterFull(bill, gw, stripe.Webhook{Secret: "whsec_x"},
		[]customer.ProductOffer{{ID: 11, MonthlyFee: 158.0, Status: "PUBLISHED"}}, ord, nil)

	body := `{"successUrl":"https://app.example/ok","cancelUrl":"https://app.example/cancel"}`
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-checkout", body, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) || data["checkoutUrl"] != "https://checkout.example/cs_t" ||
		data["sessionId"] != "cs_t" {
		t.Fatalf("checkout resp=%s", w.Body.String())
	}
	if gw.lastCents != 15800 {
		t.Fatalf("gw cents=%d", gw.lastCents)
	}

	// 缺省 success/cancel URL:相对路径合法(走 /pay/stripe/done 静态页)。
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-checkout", `{}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeOK) {
		t.Fatalf("default url resp=%s", w.Body.String())
	}

	// 非法 URL:42200。
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/orders/ORD-9/stripe-checkout",
		`{"successUrl":"ftp://bad","cancelUrl":"https://ok"}`, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeInvalidParam) {
		t.Fatalf("bad url resp=%s", w.Body.String())
	}
}
