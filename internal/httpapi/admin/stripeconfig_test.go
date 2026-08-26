package adminapi

// 契约:GET/PUT /stripe-config 与 POST /stripe-config/channel/test(menu:stripeconfig)。
// secret 字段:落库密文(enc:v1:)、GET 掩码、PUT 空串跳过;自检探活经注入点隔离真实网络。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestStripeConfigRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("GET 空配置回默认值,secret 掩码", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := getJSON(t, r, "/api/admin/v1/stripe-config", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Data struct {
				Fields map[string]struct {
					Value    string `json:"value"`
					HasValue bool   `json:"hasValue"`
				} `json:"fields"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Data.Fields["stripe.currency"].Value != "php" {
			t.Fatalf("default missing: %+v", env.Data.Fields)
		}
		if env.Data.Fields["stripe.apiKey"].Value != "" || env.Data.Fields["stripe.apiKey"].HasValue {
			t.Fatalf("secret should be masked-empty")
		}
	})

	t.Run("PUT channel/webhook secret 加密落库 空串跳过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}}
		r := newAuthTestRouter(f, mgr)
		w := putAuth(t, r, "/api/admin/v1/stripe-config/channel",
			`{"values":{"stripe.apiKey":"sk_test_x","stripe.currency":"usd","stripe.publishableKey":"pk_test_x"}}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		stored := f.params["stripe.apiKey"]
		if !strings.HasPrefix(stored, "enc:v1:") || strings.Contains(stored, "sk_test_x") {
			t.Fatalf("secret not encrypted: %q", stored)
		}
		if f.params["stripe.currency"] != "usd" || f.params["stripe.publishableKey"] != "pk_test_x" {
			t.Fatalf("plain fields not saved: %+v", f.params)
		}
		w2 := putAuth(t, r, "/api/admin/v1/stripe-config/webhook",
			`{"values":{"stripe.webhookSecret":"whsec_x"}}`, token)
		if w2.Code != 200 {
			t.Fatalf("webhook group status=%d body=%s", w2.Code, w2.Body.String())
		}
		if !strings.HasPrefix(f.params["stripe.webhookSecret"], "enc:v1:") {
			t.Fatalf("whsec not encrypted: %q", f.params["stripe.webhookSecret"])
		}
		w3 := putAuth(t, r, "/api/admin/v1/stripe-config/channel",
			`{"values":{"stripe.apiKey":""}}`, token)
		if w3.Code != 200 || f.params["stripe.apiKey"] != stored {
			t.Fatalf("empty secret should be skipped")
		}
	})

	t.Run("PUT 越组 key / 未知 group 拒绝", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := putAuth(t, r, "/api/admin/v1/stripe-config/channel",
			`{"values":{"stripe.webhookSecret":"x"}}`, token)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":42200`) {
			t.Fatalf("cross-group accepted: %d %s", w.Code, w.Body.String())
		}
		w2 := putAuth(t, r, "/api/admin/v1/stripe-config/other",
			`{"values":{"stripe.apiKey":"x"}}`, token)
		if w2.Code != 200 || !strings.Contains(w2.Body.String(), `"code":42200`) {
			t.Fatalf("unknown group accepted: %d %s", w2.Code, w2.Body.String())
		}
	})

	t.Run("test 自检:缺必填报错/草稿补全探活通过/未启用跳过", func(t *testing.T) {
		orig := stripeProbe
		stripeProbe = func(context.Context, map[string]string) error { return nil }
		defer func() { stripeProbe = orig }()
		origEP := stripeEndpointCheckFunc
		stripeEndpointCheckFunc = func(context.Context, map[string]string) string { return "" }
		defer func() { stripeEndpointCheckFunc = origEP }()

		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{}}
		r := newAuthTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/stripe-config/channel/test", `{}`, token)
		if !strings.Contains(w.Body.String(), `"ok":false`) {
			t.Fatalf("missing fields should fail: %s", w.Body.String())
		}
		// 先存 webhookSecret(webhook 组),再以 channel 草稿补 apiKey → 完整性通过 → 探活 ok
		w0 := putAuth(t, r, "/api/admin/v1/stripe-config/webhook",
			`{"values":{"stripe.webhookSecret":"whsec_x"}}`, token)
		if w0.Code != 200 {
			t.Fatalf("seed whsec failed")
		}
		w2 := postJSONAuth(t, r, "/api/admin/v1/stripe-config/channel/test",
			`{"values":{"stripe.apiKey":"sk_test_x"}}`, token)
		if !strings.Contains(w2.Body.String(), `"ok":true`) {
			t.Fatalf("draft probe should pass: %s", w2.Body.String())
		}
		w3 := putAuth(t, r, "/api/admin/v1/stripe-config/channel", `{"values":{"stripe.enabled":"false"}}`, token)
		if w3.Code != 200 {
			t.Fatalf("disable failed")
		}
		w4 := postJSONAuth(t, r, "/api/admin/v1/stripe-config/channel/test", `{}`, token)
		if !strings.Contains(w4.Body.String(), "跳过自检") {
			t.Fatalf("disabled should skip: %s", w4.Body.String())
		}
	})
}

// 探活失败路径:注入错误 → ok:false 且带原因。
func TestStripeConfigTestProbeFail(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	orig := stripeProbe
	stripeProbe = func(context.Context, map[string]string) error { return errProbeBoom }
	defer func() { stripeProbe = orig }()

	f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{
		"stripe.apiKey":        "sk_test_x",
		"stripe.webhookSecret": "whsec_x",
	}}
	r := newAuthTestRouter(f, mgr)
	w := postJSONAuth(t, r, "/api/admin/v1/stripe-config/channel/test", `{}`, token)
	if !strings.Contains(w.Body.String(), `"ok":false`) || !strings.Contains(w.Body.String(), "探活失败") {
		t.Fatalf("probe fail should surface: %s", w.Body.String())
	}
}

// 跨组草稿自检(P2-2):apiKey(webhook 组)与 webhookSecret(webhook 组)都仅以草稿提交
// 未落库,自检也应通过(草稿跨两组合并);endpoint 一致性检查消息透出。
func TestStripeConfigTestCrossGroupDraft(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	orig := stripeProbe
	stripeProbe = func(context.Context, map[string]string) error { return nil }
	defer func() { stripeProbe = orig }()
	origEP := stripeEndpointCheckFunc
	stripeEndpointCheckFunc = func(_ context.Context, cur map[string]string) string {
		if cur["stripe.webhookSecret"] != "whsec_draft" {
			return "draft-missing"
		}
		return "。警告:后台 webhook endpoint URL 与期望不一致"
	}
	defer func() { stripeEndpointCheckFunc = origEP }()

	f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{}}
	r := newAuthTestRouter(f, mgr)
	// webhookSecret 属于 webhook 组,仅作草稿(不 PUT)提交,apiKey 同理草稿。
	w := postJSONAuth(t, r, "/api/admin/v1/stripe-config/channel/test",
		`{"values":{"stripe.apiKey":"sk_test_x","stripe.webhookSecret":"whsec_draft"}}`, token)
	if !strings.Contains(w.Body.String(), `"ok":true`) || !strings.Contains(w.Body.String(), "后台 webhook endpoint URL 与期望不一致") {
		t.Fatalf("cross-group draft should pass and surface endpoint warn: %s", w.Body.String())
	}
}

// endpoint 一致性:期望 URL 未配置时自检透出提示(提示而非失败)。
func TestStripeConfigTestEndpointCheckNoWant(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	orig := stripeProbe
	stripeProbe = func(context.Context, map[string]string) error { return nil }
	defer func() { stripeProbe = orig }()

	f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{
		"stripe.apiKey": "sk_test_x", "stripe.webhookSecret": "whsec_x",
	}}
	r := newAuthTestRouter(f, mgr)
	w := postJSONAuth(t, r, "/api/admin/v1/stripe-config/channel/test", `{}`, token)
	if !strings.Contains(w.Body.String(), "未配置期望回调 URL") {
		t.Fatalf("no-want hint should surface: %s", w.Body.String())
	}
}

var errProbeBoom = errors.New("probe boom")
