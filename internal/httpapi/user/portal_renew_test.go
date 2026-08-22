package userapi

// 套餐续费端点测试:收款金额/N月x月费、赠送阶梯命中、合约延长、未命中/无券路径。

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func newRenewRouter(ud *fakeUserData, promoSvc *fakePromo, bill *fakeBilling) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	r := gin.New()
	Register(r, &app.Application{
		Customer: &userPortalCustSvc{c: userPortalCust()},
		Billing:  bill, Portal: portal.NewMemory(),
		UserData: ud, Promotion: promoSvc,
	}, mgr)
	return r, mgr
}

// TestPortal_RenewWithGift 购 12 月命中"12送3":收款 12x月费,合约 +15 月,赠送落痕。
func TestPortal_RenewWithGift(t *testing.T) {
	ud := &fakeUserData{renewProductID: 5, renewFee: 100, renewEnd: "2027-08"}
	promoSvc := &fakePromo{match: &promotion.GiftRule{RuleID: 3, BuyMonths: 12, GiftMonths: 3}}
	bill := &fakeBilling{}
	r, mgr := newRenewRouter(ud, promoSvc, bill)
	tok, _ := signCustomerToken(mgr, 7, "13800001234")

	out := portalDo(r, http.MethodPost, "/api/user/v1/plans/9/renew",
		`{"months":12,"payMethod":"wechat"}`, tok)
	if code := out["code"]; code != float64(0) {
		t.Fatalf("renew resp=%v", out)
	}
	data := out["data"].(map[string]any)
	if data["giftMonths"] != float64(3) || data["buyMonths"] != float64(12) {
		t.Fatalf("months resp=%v", data)
	}
	if data["amount"] != float64(1200) || data["contractEnd"] != "2027-08" {
		t.Fatalf("amount/end resp=%v", data)
	}
	if ud.renewedMonths != 15 {
		t.Fatalf("renewedMonths=%d want 15", ud.renewedMonths)
	}
	if len(bill.pays) != 1 || bill.pays[0].Amount != 1200 || bill.pays[0].CouponID != "" {
		t.Fatalf("pays=%+v", bill.pays)
	}
	if len(promoSvc.recorded) != 1 ||
		promoSvc.recorded[0].GiftMonths != 3 || promoSvc.recorded[0].PaymentID != 901 {
		t.Fatalf("recorded=%+v", promoSvc.recorded)
	}
}

// TestPortal_RenewNoGift 无命中阶梯:赠送 0,合约仅 +N 月,无落痕。
func TestPortal_RenewNoGift(t *testing.T) {
	ud := &fakeUserData{renewProductID: 5, renewFee: 100, renewEnd: "2026-10"}
	promoSvc := &fakePromo{}
	r, mgr := newRenewRouter(ud, promoSvc, &fakeBilling{})
	tok, _ := signCustomerToken(mgr, 7, "13800001234")

	out := portalDo(r, http.MethodPost, "/api/user/v1/plans/9/renew",
		`{"months":6,"payMethod":"alipay"}`, tok)
	data, _ := out["data"].(map[string]any)
	if out["code"] != float64(0) || data["giftMonths"] != float64(0) || ud.renewedMonths != 6 {
		t.Fatalf("resp=%v renewed=%d", out, ud.renewedMonths)
	}
	if len(promoSvc.recorded) != 0 {
		t.Fatalf("recorded=%+v", promoSvc.recorded)
	}
}

// TestPortal_RenewPlanNotFound 套餐未命中/非本人:404。
func TestPortal_RenewPlanNotFound(t *testing.T) {
	r, mgr := newRenewRouter(&fakeUserData{}, &fakePromo{}, &fakeBilling{})
	tok, _ := signCustomerToken(mgr, 7, "13800001234")
	out := portalDo(r, http.MethodPost, "/api/user/v1/plans/9/renew",
		`{"months":6,"payMethod":"wechat"}`, tok)
	if out["code"] != float64(40400) && out["code"] != float64(404) {
		t.Fatalf("resp=%v", out)
	}
}
