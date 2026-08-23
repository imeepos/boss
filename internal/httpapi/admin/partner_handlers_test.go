package adminapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/partner"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

type partnerHandlerStub struct {
	partner.Service
	profile partner.PartnerProfile
	orders  []partner.OrderRow
}

func (s *partnerHandlerStub) Profile(context.Context, int64) (partner.PartnerProfile, error) {
	return s.profile, nil
}
func (s *partnerHandlerStub) ListOrders(context.Context, int64) ([]partner.OrderRow, error) {
	return s.orders, nil
}

type commissionHandlerStub struct {
	partner.CommissionLedgerService
}

func (commissionHandlerStub) ListCommissionLedger(context.Context, int64, string) ([]partner.CommissionLedgerRow, error) {
	return []partner.CommissionLedgerRow{{OrderID: 11, Status: partner.CommissionAccrued}}, nil
}
func (commissionHandlerStub) AccrueCommission(context.Context, int64, int64, float64, float64) (int64, error) {
	return 0, nil
}
func (commissionHandlerStub) SettleCommissionLedger(context.Context, int64, int64) error { return nil }

type auditHandlerStub struct{ partner.AuditReportService }

func (auditHandlerStub) ListAuditReport(context.Context, int64, string) ([]partner.AuditReportRow, error) {
	return []partner.AuditReportRow{{Action: "partner_order.submit", Count: 3}}, nil
}

type regionHandlerStub struct{ partner.RegionScopeService }

func (regionHandlerStub) GetRegionScope(context.Context, int64) (partner.RegionScope, error) {
	return partner.RegionScope{LegalEntityID: 9, RegionPath: "root.luzon"}, nil
}
func (regionHandlerStub) SetRegionScope(context.Context, int64, string) error { return nil }

func partnerTestContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(middleware.CtxClaims, &auth.Claims{AccountID: 7})
	return c, w
}

func TestPartnerOrdersHandler(t *testing.T) {
	stub := &partnerHandlerStub{orders: []partner.OrderRow{{OrderNo: "ORD-001"}}}
	c, w := partnerTestContext(http.MethodGet, "/partner/orders", "")
	partnerOrdersHandler(&app.Application{Partner: stub})(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestPartnerCommissionHandler(t *testing.T) {
	c, w := partnerTestContext(http.MethodGet, "/partner/commissions", "")
	partnerCommissionListHandler(&app.Application{PartnerCommission: commissionHandlerStub{}})(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestPartnerAuditHandler(t *testing.T) {
	c, w := partnerTestContext(http.MethodGet, "/partner/audit-report", "")
	partnerAuditReportHandler(&app.Application{PartnerAudit: auditHandlerStub{}})(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestPartnerRegionScopeHandlers(t *testing.T) {
	stub := regionHandlerStub{}
	a := &app.Application{PartnerRegion: stub}
	c, w := partnerTestContext(http.MethodGet, "/partner/region-scope", "")
	partnerRegionScopeHandler(a)(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", w.Code, w.Body.String())
	}
	c, w = partnerTestContext(http.MethodPut, "/partner/region-scope", `{"regionPath":"root.luzon.ncr"}`)
	partnerRegionScopeUpdateHandler(a)(c)
	if w.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", w.Code, w.Body.String())
	}
}
