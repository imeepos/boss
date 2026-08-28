// 支付方式下发 handler 测试:stripe 未配置 → 无银行卡、默认线下收款;已配置 → 含银行卡、默认卡。
package userapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// portalMethodsGet 直接挂载 handler 断言(端点不依赖客户身份,绕过认证组)。
func portalMethodsGet(t *testing.T, a *app.Application) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/payments/methods", portalPayMethods(a))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/payments/methods", nil))
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

// TestPortalPayMethods_StripeUnconfigured stripe 未配置:默认 cash 线下收款,不含银行卡。
func TestPortalPayMethods_StripeUnconfigured(t *testing.T) {
	out := portalMethodsGet(t, &app.Application{})
	data := out["data"].(map[string]any)
	if data["default"] != "cash" {
		t.Fatalf("default want cash(线下收款), got %v", data["default"])
	}
	items := data["items"].([]any)
	for _, it := range items {
		if it.(map[string]any)["key"] == "card" {
			t.Fatalf("stripe 未配置不应下发银行卡: %v", items)
		}
	}
	if len(items) != 3 {
		t.Fatalf("want wechat/alipay/cash 三项, got %v", items)
	}
}

// TestPortalPayMethods_StripeReady stripe 已配置:默认 card,含银行卡。
func TestPortalPayMethods_StripeReady(t *testing.T) {
	a := &app.Application{Stripe: stripe.NewDynamic(func(context.Context) (stripe.Config, error) {
		return stripe.Config{Enabled: true, APIKey: "sk_test_x"}, nil
	})}
	out := portalMethodsGet(t, a)
	data := out["data"].(map[string]any)
	if data["default"] != "card" {
		t.Fatalf("default want card, got %v", data["default"])
	}
	items := data["items"].([]any)
	if len(items) != 4 {
		t.Fatalf("want 4 items incl card, got %v", items)
	}
	last := items[len(items)-1].(map[string]any)
	if last["key"] != "card" || last["label"] != "银行卡" {
		t.Fatalf("card item mismatch: %v", last)
	}
}
