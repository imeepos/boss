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
