package stripe

// endpoint 管理契约:列表/更新/创建;创建响应携带一次性 secret。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestListWebhookEndpoints 列表:取 id/url;非本路径 endpoint 一并返回由调用方过滤。
func TestListWebhookEndpoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "we_1", "url": "https://a.example.com/api/user/v1/webhooks/stripe"},
				{"id": "we_2", "url": "https://b.example.com/other"},
			},
		})
	}))
	defer srv.Close()
	c := &Client{APIKey: "sk", BaseURL: srv.URL, HTTP: srv.Client()}
	eps, err := c.ListWebhookEndpoints(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(eps) != 2 || eps[0].ID != "we_1" || eps[1].URL != "https://b.example.com/other" {
		t.Fatalf("eps=%+v", eps)
	}
}

// TestUpdateWebhookEndpoint 更新:POST /v1/webhook_endpoints/:id,form 带 url。
func TestUpdateWebhookEndpoint(t *testing.T) {
	var path, form string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		form = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"we_1","url":"https://new.example.com/path"}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "sk", BaseURL: srv.URL, HTTP: srv.Client()}
	want := "https://new.example.com/api/user/v1/webhooks/stripe"
	if err := c.UpdateWebhookEndpoint(context.Background(), "we_1", want); err != nil {
		t.Fatalf("update: %v", err)
	}
	if path != "/v1/webhook_endpoints/we_1" || !strings.Contains(form, "url=") {
		t.Fatalf("path=%s form=%s", path, form)
	}
}

// TestCreateWebhookEndpoint 创建:form 带 url + enabled_events,响应解析 secret。
func TestCreateWebhookEndpoint(t *testing.T) {
	var path, form, idem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		idem = r.Header.Get("Idempotency-Key")
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		form = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"we_new","url":"https://x.example.com/p","secret":"whsec_new"}`))
	}))
	defer srv.Close()
	c := &Client{APIKey: "sk", BaseURL: srv.URL, HTTP: srv.Client()}
	ep, secret, err := c.CreateWebhookEndpoint(context.Background(),
		"https://x.example.com/api/user/v1/webhooks/stripe",
		[]string{"payment_intent.succeeded", "payment_intent.payment_failed"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if ep.ID != "we_new" || secret != "whsec_new" {
		t.Fatalf("ep=%+v secret=%s", ep, secret)
	}
	if path != "/v1/webhook_endpoints" || idem == "" ||
		!strings.Contains(form, "enabled_events%5B%5D=payment_intent.succeeded") {
		t.Fatalf("path=%s idem=%s form=%s", path, idem, form)
	}
}
