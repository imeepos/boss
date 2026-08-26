package app

// Stripe webhook 自愈循环契约:URL 失配自动 UPDATE endpoint;缺失自动 CREATE 并持久化
// whsec;健康时无副作用;自检失败/重建都进提醒中心告警。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// fakeGuardParams 桩 biz_params 读写。
type fakeGuardParams struct {
	stored  map[string]string
	updates []string
}

func (f *fakeGuardParams) ListParams(context.Context) ([]user.Param, error) {
	params := make([]user.Param, 0, len(f.stored))
	for k, v := range f.stored {
		params = append(params, user.Param{Key: k, Value: v})
	}
	return params, nil
}

func (f *fakeGuardParams) UpdateParam(_ context.Context, key, value string, _ int64) error {
	f.updates = append(f.updates, key+"="+value)
	if f.stored == nil {
		f.stored = map[string]string{}
	}
	f.stored[key] = value
	return nil
}

// guardStripeServer 模拟 Stripe endpoint API;endpoints 由调用方在 handler 里改。
func guardStripeServer(reg *[]stripe.WebhookEndpoint) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/webhook_endpoints":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": *reg})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/webhook_endpoints/"):
			// 更新:记录在 body 中回显(调用方无需校验内容)
			_ = r.ParseForm()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"we_1","url":"` + r.FormValue("url") + `"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/webhook_endpoints":
			_ = r.ParseForm()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"we_new","url":"` + r.FormValue("url") + `","secret":"whsec_new"}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}

// newGuardDeps 组装自愈循环依赖:resolver 指向 httptest Stripe,期望 URL 固定。
func newGuardDeps(reg *[]stripe.WebhookEndpoint, wantURL string) (stripeGuardDeps, *fakeGuardParams, *notify.MemStore) {
	srv := guardStripeServer(reg)
	dyn := stripe.NewDynamic(func(context.Context) (stripe.Config, error) {
		return stripe.Config{
			Enabled: true, APIKey: "sk_test_x", APIBaseURL: srv.URL, WebhookURL: wantURL,
		}, nil
	})
	params := &fakeGuardParams{stored: map[string]string{}}
	n := notify.NewMemStore()
	return stripeGuardDeps{stripe: dyn, params: params, n: n}, params, n
}

func guardNotifyTitles(n *notify.MemStore) []string {
	items, _, _ := n.List(context.Background(), "", 0, notify.Filter{})
	titles := make([]string, 0, len(items))
	for _, it := range items {
		titles = append(titles, it.Title)
	}
	return titles
}

// 健康:注册 URL 与期望一致 → 无更新/创建/告警。
func TestStripeWebhookGuard_HealthyNoop(t *testing.T) {
	want := "https://t.example.com/api/user/v1/webhooks/stripe"
	reg := &[]stripe.WebhookEndpoint{{ID: "we_1", URL: want}}
	d, params, n := newGuardDeps(reg, want)
	runStripeWebhookOnce(context.Background(), d)
	if len(params.updates) != 0 || len(guardNotifyTitles(n)) != 0 {
		t.Fatalf("healthy should be noop: updates=%v titles=%v", params.updates, guardNotifyTitles(n))
	}
}

// 失配:注册 URL 旧(隧道已变)→ UPDATE endpoint,告警自愈;无 whsec 变更。
func TestStripeWebhookGuard_URLMismatchHeals(t *testing.T) {
	want := "https://new.example.com/api/user/v1/webhooks/stripe"
	old := "https://old.example.com/api/user/v1/webhooks/stripe"
	reg := &[]stripe.WebhookEndpoint{{ID: "we_1", URL: old}}
	d, params, n := newGuardDeps(reg, want)
	runStripeWebhookOnce(context.Background(), d)
	titles := guardNotifyTitles(n)
	if len(titles) != 1 || !strings.Contains(titles[0], "已自愈") {
		t.Fatalf("titles=%v", titles)
	}
	if len(params.updates) != 0 {
		t.Fatalf("update should not touch whsec: %v", params.updates)
	}
}

// 缺失:无本系统 endpoint → CREATE,新 whsec 落 biz_params,告警重建。
func TestStripeWebhookGuard_NoEndpointCreates(t *testing.T) {
	want := "https://new.example.com/api/user/v1/webhooks/stripe"
	reg := &[]stripe.WebhookEndpoint{{ID: "we_x", URL: "https://other.example.com/not-ours"}}
	d, params, n := newGuardDeps(reg, want)
	runStripeWebhookOnce(context.Background(), d)
	titles := guardNotifyTitles(n)
	if len(titles) != 1 || !strings.Contains(titles[0], "已重建") {
		t.Fatalf("titles=%v", titles)
	}
	found := false
	for _, u := range params.updates {
		if strings.HasPrefix(u, "stripe.webhookSecret=enc:v1:") {
			found = true
		}
	}
	if !found {
		t.Fatalf("whsec not persisted: %v", params.updates)
	}
}

// 未配置期望 URL:跳过,无副作用(隧道未 feed 时不误告警)。
func TestStripeWebhookGuard_NoWantSkips(t *testing.T) {
	reg := &[]stripe.WebhookEndpoint{{ID: "we_1", URL: "https://stale.example.com/api/user/v1/webhooks/stripe"}}
	srv := guardStripeServer(reg)
	dyn := stripe.NewDynamic(func(context.Context) (stripe.Config, error) {
		return stripe.Config{Enabled: true, APIKey: "sk_test_x", APIBaseURL: srv.URL, WebhookURL: ""}, nil
	})
	d := stripeGuardDeps{stripe: dyn, params: &fakeGuardParams{}, n: notify.NewMemStore()}
	runStripeWebhookOnce(context.Background(), d)
	if len(guardNotifyTitles(d.n.(*notify.MemStore))) != 0 {
		t.Fatalf("no-want should skip silently")
	}
}
