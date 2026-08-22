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
func (f *settleBilling) CreateBill(context.Context, billing.Bill) (int64, error) { return 0, nil }
func (f *settleBilling) GetBill(context.Context, int64) (*billing.Bill, error)   { return nil, nil }
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

// newStripeRouter gw 为 nil 表示通道未配置(空注册表)。
func newStripeRouter(bill billing.BillingService, gw billing.PaymentGateway, wh stripe.Webhook) *gin.Engine {
	gin.SetMode(gin.TestMode)
	a := &app.Application{
		Customer: &userPortalCustSvc{c: userPortalCust()}, Billing: bill,
		Portal: portal.NewMemory(), StripeWebhook: wh,
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

// TestStripeWebhookTopup 无账单意图(customer_id 有、bill_no 空):落充值流水。
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
}
