package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCreateIntent 契约:Bearer 鉴权 + form 参数(amount/currency/metadata)+ 幂等键。
func TestCreateIntent(t *testing.T) {
	var gotAuth, gotIDem, gotForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotIDem = r.Header.Get("Idempotency-Key")
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotForm = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "pi_123", "client_secret": "pi_123_secret", "status": "requires_confirmation",
			"amount": 9900, "currency": "php",
		})
	}))
	defer srv.Close()

	c := &Client{APIKey: "sk_test_x", BaseURL: srv.URL, Currency: "php", HTTP: srv.Client()}
	it, err := c.CreateIntent(context.Background(), "PAY-1", 9900, map[string]string{"bill_no": "B-9", "customer_id": "7"})
	if err != nil {
		t.Fatalf("CreateIntent: %v", err)
	}
	if it.ID != "pi_123" || it.ClientSecret != "pi_123_secret" || it.AmountCents != 9900 {
		t.Fatalf("intent: %+v", it)
	}
	if gotAuth != "Bearer sk_test_x" || gotIDem != "PAY-1" {
		t.Fatalf("headers: auth=%q idem=%q", gotAuth, gotIDem)
	}
	for _, want := range []string{"amount=9900", "currency=php", "metadata%5Bpay_no%5D=PAY-1", "metadata%5Bbill_no%5D=B-9"} {
		if !strings.Contains(gotForm, want) {
			t.Fatalf("form missing %s: %s", want, gotForm)
		}
	}
}

// TestCreateIntentError 非法金额等业务错:Stripe 返回 4xx,错误带状态与正文。
func TestCreateIntentError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"amount too small"}}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "sk", BaseURL: srv.URL, Currency: "php", HTTP: srv.Client()}
	if _, err := c.CreateIntent(context.Background(), "PAY-2", 1, nil); err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("err: %v", err)
	}
}

// TestCreateCheckoutSession 契约:mode/success_url/line_items 单价与 metadata 透传。
func TestCreateCheckoutSession(t *testing.T) {
	var gotForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		gotForm = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cs_1","url":"https://checkout.stripe.com/c/pay/cs_1"}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "sk", BaseURL: srv.URL, Currency: "php", HTTP: srv.Client()}
	s, err := c.CreateCheckoutSession(context.Background(), "PAY-3", 9900,
		map[string]string{"bill_no": "B-1"}, "https://app.example/pay/ok", "https://app.example/pay/cancel")
	if err != nil || s.ID != "cs_1" || !strings.Contains(s.URL, "cs_1") {
		t.Fatalf("session: %+v err=%v", s, err)
	}
	for _, want := range []string{"mode=payment", "success_url=https%3A%2F%2Fapp.example%2Fpay%2Fok",
		"cancel_url=", "unit_amount%5D=9900", "currency%5D=php", "metadata%5Bpay_no%5D=PAY-3", "metadata%5Bbill_no%5D=B-1",
		// 关键:Session 级 metadata 必须同时透传到 PaymentIntent(payment_intent.succeeded 事件只带 PI)
		"payment_intent_data%5Bmetadata%5D%5Bpay_no%5D=PAY-3", "payment_intent_data%5Bmetadata%5D%5Bbill_no%5D=B-1"} {
		if !strings.Contains(gotForm, want) {
			t.Fatalf("form missing %s: %s", want, gotForm)
		}
	}
}

// TestListDayStatements 契约:按日窗口拉流水,仅取 charge 行,metadata.pay_no 对齐;
// 分页 has_more 续拉。
func TestListDayStatements(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/v1/balance_transactions" {
			t.Fatalf("req: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk" {
			t.Fatalf("auth missing")
		}
		q := r.URL.Query()
		body := `{"has_more":false,"data":[]}`
		if calls == 1 {
			if q.Get("expand[]") != "data.source.payment_intent" || q.Get("created[gte]") == "" {
				t.Fatalf("page1 params: %v", q)
			}
			body = `{"has_more":true,"data":[
				{"id":"txn_1","amount":9900,"source":{"object":"charge",
				 "payment_intent":{"metadata":{"pay_no":"PAY-A"}}}},
				{"id":"txn_2","amount":-100,"source":{"object":"fee"}},
				{"id":"txn_3","amount":5000,"source":{"object":"charge","payment_intent":{"metadata":{}}}}
			]}`
		} else if q.Get("starting_after") != "txn_3" {
			t.Fatalf("page2 cursor: %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	c := &Client{APIKey: "sk", BaseURL: srv.URL, Currency: "php", HTTP: srv.Client()}

	rows, err := c.ListDayStatements(context.Background(), time.Date(2026, 8, 23, 15, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatalf("ListDayStatements: %v", err)
	}
	if calls != 2 || len(rows) != 2 { // fee 行不入比对
		t.Fatalf("calls=%d rows=%d", calls, len(rows))
	}
	if rows[0].PayNo != "PAY-A" || rows[0].Amount != 99 {
		t.Fatalf("row0: %+v", rows[0])
	}
	if rows[1].PayNo != "" || rows[1].Amount != 50 {
		t.Fatalf("rows: %+v", rows)
	}
}

// TestNewEmptyKey 契约:无密钥返回 nil(装配层判空降级)。
func TestNewEmptyKey(t *testing.T) {
	if New("", "php") != nil {
		t.Fatal("New with empty key should be nil")
	}
}

// TestWebhookVerify 签名往返:正确签名过、篡改拒绝、过期时间戳拒绝。
func TestWebhookVerify(t *testing.T) {
	w := Webhook{Secret: "whsec_x"}
	payload := []byte(`{"type":"payment_intent.succeeded"}`)
	sig := w.SignPayload(payload, time.Now())
	if err := w.Verify(payload, sig); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := w.Verify([]byte(`tampered`), sig); err != ErrBadSignature {
		t.Fatalf("tampered: %v", err)
	}
	if err := w.Verify(payload, w.SignPayload(payload, time.Now().Add(-10*time.Minute))); err != ErrStaleTS {
		t.Fatalf("stale: %v", err)
	}
	if err := w.Verify(payload, "v1=deadbeef"); err != ErrBadSignature {
		t.Fatalf("malformed header: %v", err)
	}
}

// TestParseEvent 契约:抽 type/intent/amount + metadata(pay_no/bill_no/customer_id)。
func TestParseEvent(t *testing.T) {
	ev, err := ParseEvent([]byte(`{
		"type":"payment_intent.succeeded",
		"data":{"object":{"id":"pi_9","amount":15800,"currency":"php",
		"metadata":{"pay_no":"PAY-20260822-01","bill_no":"B-2026-08","customer_id":"7"}}}
	}`))
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if ev.Type != "payment_intent.succeeded" || ev.PayNo != "PAY-20260822-01" ||
		ev.BillNo != "B-2026-08" || ev.CustomerID != 7 || ev.AmountCents != 15800 || ev.IntentID != "pi_9" {
		t.Fatalf("event: %+v", ev)
	}
}

// TestProbeBalance 自检探活:2xx 通过;4xx/5xx 报错带状态。
func TestProbeBalance(t *testing.T) {
	status := 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if status == 200 {
			w.Write([]byte(`{"available":[]}`))
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(`{"error":{"message":"invalid api_key"}}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "sk_test_x", BaseURL: srv.URL, HTTP: srv.Client()}
	if err := c.ProbeBalance(context.Background()); err != nil {
		t.Fatalf("probe 200: %v", err)
	}
	status = 401
	if err := c.ProbeBalance(context.Background()); err == nil {
		t.Fatal("probe 401 should error")
	}
}
