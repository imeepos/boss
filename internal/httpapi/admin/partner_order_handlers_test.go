package adminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/partner"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

type partnerOrderStub struct {
	partner.Service
	profile partner.PartnerProfile
}

func (s partnerOrderStub) Profile(context.Context, int64) (partner.PartnerProfile, error) {
	return s.profile, nil
}

type orderSubmitStub struct {
	order.OrderService
	got    order.SubmitReq
	result *order.Order
}

func (s *orderSubmitStub) Submit(_ context.Context, req order.SubmitReq) (*order.Order, error) {
	s.got = req
	return s.result, nil
}

type orderRiskStub struct{ partner.OrderRiskDecision }

func (s orderRiskStub) CheckOrderRisk(context.Context, int64, int64) (partner.OrderRiskDecision, error) {
	return s.OrderRiskDecision, nil
}

func TestPartnerOrderSubmitMarksPartnerAndTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orders := &orderSubmitStub{result: &order.Order{ID: 11, OrderNo: "ORD-P-1", Stage: 1, Status: "PENDING"}}
	a := &app.Application{
		Partner:          &partnerOrderStub{profile: partner.PartnerProfile{LegalEntityID: 9}},
		Order:            orders,
		PartnerOrderRisk: orderRiskStub{OrderRiskDecision: partner.OrderRiskDecision{Allowed: true}},
	}
	c, w := partnerOrderTestContext(http.MethodPost, "/partner/orders")
	c.Request = httptest.NewRequest(http.MethodPost, "/partner/orders", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Body = http.NoBody
	c.Request = httptest.NewRequest(http.MethodPost, "/partner/orders", strings.NewReader(`{"customerId":1,"offerId":2,"addressId":3,"channelId":4}`))
	c.Request.Header.Set("Content-Type", "application/json")
	partnerOrderSubmitHandler(a)(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !orders.got.PartnerOrder || orders.got.LegalEntityID != 9 {
		t.Fatalf("request=%+v", orders.got)
	}
}

func TestPartnerOrderSubmitBlocksRisk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	orders := &orderSubmitStub{}
	a := &app.Application{
		Partner: &partnerOrderStub{profile: partner.PartnerProfile{LegalEntityID: 9}}, Order: orders,
		PartnerOrderRisk: orderRiskStub{OrderRiskDecision: partner.OrderRiskDecision{Reason: "CUSTOMER_COOLDOWN"}},
	}
	c, w := partnerOrderTestContext(http.MethodPost, "/partner/orders")
	c.Request = httptest.NewRequest(http.MethodPost, "/partner/orders", strings.NewReader(`{"customerId":1,"offerId":2,"addressId":3,"channelId":4}`))
	c.Request.Header.Set("Content-Type", "application/json")
	partnerOrderSubmitHandler(a)(c)
	if w.Code != http.StatusTooManyRequests && w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if orders.got.CustomerID != 0 {
		t.Fatal("order service called after risk block")
	}
}

func partnerOrderTestContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Set(middleware.CtxClaims, &auth.Claims{AccountID: 7})
	return c, w
}
