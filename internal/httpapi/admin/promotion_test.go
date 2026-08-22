package adminapi

// 促销域路由冒烟:模板创建/发放/兑换码/赠送规则(权限门 + envelope 契约)。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakePromo 桩 promotion.Service。
type fakePromo struct {
	tpl    promotion.Template
	issued int
	codes  []promotion.CouponCode
}

func (f *fakePromo) ListTemplates(context.Context) ([]promotion.Template, error) {
	return []promotion.Template{f.tpl}, nil
}
func (f *fakePromo) CreateTemplate(_ context.Context, t promotion.Template) (int64, error) {
	f.tpl = t
	return 5, nil
}
func (f *fakePromo) DisableTemplate(context.Context, int64) error { return nil }
func (f *fakePromo) Issue(context.Context, int64, []int64) (int, error) {
	return f.issued, nil
}
func (f *fakePromo) IssueToCustomer(context.Context, int64, int64, string) (string, error) {
	return "CPN-x", nil
}
func (f *fakePromo) PointsPrice(context.Context, int64) (int64, error) { return 0, nil }
func (f *fakePromo) CreateCodes(_ context.Context, _ int64, n int) ([]promotion.CouponCode, error) {
	return f.codes[:n], nil
}
func (f *fakePromo) ListCodes(context.Context, int64) ([]promotion.CouponCode, error) {
	return f.codes, nil
}
func (f *fakePromo) RedeemCode(context.Context, string, int64) (string, error) {
	return "", nil
}
func (f *fakePromo) CreateGift(context.Context, string, int64) (string, error) {
	return "GFT-x", nil
}
func (f *fakePromo) ListCustomerCoupons(context.Context, int64, string, int64) ([]map[string]any, error) {
	return nil, nil
}
func (f *fakePromo) RollbackRedemption(context.Context, int64) error { return nil }
func (f *fakePromo) ListGiftRules(context.Context) ([]promotion.GiftRule, error) {
	return nil, nil
}
func (f *fakePromo) CreateGiftRule(context.Context, promotion.GiftRule) (int64, error) {
	return 9, nil
}
func (f *fakePromo) DisableGiftRule(context.Context, int64) error { return nil }
func (f *fakePromo) MatchGiftRule(context.Context, int64, int) (*promotion.GiftRule, error) {
	return nil, nil
}
func (f *fakePromo) RecordGift(context.Context, promotion.GiftRecord) error { return nil }

func TestPromotionRoutesSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	promo := &fakePromo{issued: 2,
		codes: []promotion.CouponCode{{CodeID: 1, Code: "RDM-1", Status: "UNUSED"}}}
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: true}, Promotion: promo}, mgr)
	tok := authToken(t, mgr)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	if w := do(http.MethodPost, "/api/admin/v1/coupon-templates",
		`{"legalEntityId":1,"name":"开户满减","type":"FULL_CUT","faceValue":2000,"threshold":10000}`); w.Code != http.StatusOK {
		t.Fatalf("create tpl: %d %s", w.Code, w.Body.String())
	}
	if w := do(http.MethodPost, "/api/admin/v1/coupon-templates/5/issue",
		`{"customerIds":[7,8]}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"issued":2`) {
		t.Fatalf("issue: %d %s", w.Code, w.Body.String())
	}
	if w := do(http.MethodPost, "/api/admin/v1/coupon-templates/5/codes",
		`{"count":1}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "RDM-1") {
		t.Fatalf("codes: %d %s", w.Code, w.Body.String())
	}
	if w := do(http.MethodPost, "/api/admin/v1/gift-rules",
		`{"legalEntityId":1,"name":"12送3","buyMonths":12,"giftMonths":3}`); w.Code != http.StatusOK {
		t.Fatalf("gift rule: %d %s", w.Code, w.Body.String())
	}

	// 无权限 403
	r2 := gin.New()
	Register(r2, &app.Application{User: &fakeUser{permOk: false}, Promotion: &fakePromo{}}, mgr)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/coupon-templates", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("perm gate: %d", w.Code)
	}
}
