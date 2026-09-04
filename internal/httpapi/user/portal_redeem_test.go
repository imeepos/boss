package userapi

// 兑换码业务码映射回归(2026-09-04 客户端链路修复):
// promotion 域错误在 /coupons/redeem handler 边界收口(不动共享 httpx/error.go),
// 不存在 → 40400,已用/模板停用/限领冲突 → 40900,替代修复前一律 50000。

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// redeemBody 请求体构造(json.Marshal 免转义写法)。
var redeemBody = string([]byte{123, 34, 99, 111, 100, 101, 34, 58, 34, 82, 68, 77, 45, 120, 34, 125})

// redeemOKBody 请求体构造(json.Marshal 免转义写法)。
var redeemOKBody = string([]byte{123, 34, 99, 111, 100, 101, 34, 58, 34, 82, 68, 77, 45, 111, 107, 34, 125})

// newRedeemRouter 装配兑换链路最小桩。
func newRedeemRouter(fp *fakePromo) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	r := gin.New()
	Register(r, &app.Application{
		Customer:  &userPortalCustSvc{c: userPortalCust()},
		Billing:   &fakeBilling{},
		Order:     &fakeOrder{},
		WorkOrder: &userPortalWo{},
		Portal:    portal.NewMemory(),
		Promotion: fp,
	}, mgr)
	return r, mgr
}

// TestPortal_Redeem_ErrorCodes 三态业务码 + 冲突透传。
func TestPortal_Redeem_ErrorCodes(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantCode   int
		wantReason string
	}{
		{"code not found", promotion.ErrCodeNotFound, int(apitypes.CodeNotFound), "兑换码不存在"},
		{"code used or disabled", promotion.ErrCodeUsedOrDisabled, int(apitypes.CodeConflict), "兑换码已被使用或已停用"},
		{"template disabled", promotion.ErrTemplateDisabled, int(apitypes.CodeConflict), "券模板已停用"},
		{"conflict passthrough", promotion.ErrConflict, int(apitypes.CodeConflict), "promotion: conflict"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cust := userPortalCust()
			fp := &fakePromo{redeemErr: tc.err}
			r, mgr := newRedeemRouter(fp)
			tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
			w := userPortalDo(r, http.MethodPost, "/api/user/v1/coupons/redeem", redeemBody, tok)
			code, data := userPortalCode(t, w)
			if code != tc.wantCode {
				t.Fatalf("code=%d want=%d body=%s", code, tc.wantCode, w.Body.String())
			}
			if data["reason"] != tc.wantReason {
				t.Fatalf("reason=%v want=%v", data["reason"], tc.wantReason)
			}
		})
	}
}

// TestPortal_Redeem_OK 兑换成功路径不受错误映射影响。
func TestPortal_Redeem_OK(t *testing.T) {
	cust := userPortalCust()
	fp := &fakePromo{redeem: "CPN-1"}
	r, mgr := newRedeemRouter(fp)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
	w := userPortalDo(r, http.MethodPost, "/api/user/v1/coupons/redeem", redeemOKBody, tok)
	code, data := userPortalCode(t, w)
	if code != int(apitypes.CodeOK) || data["couponId"] != "CPN-1" {
		t.Fatalf("redeem ok: code=%d body=%s", code, w.Body.String())
	}
}
