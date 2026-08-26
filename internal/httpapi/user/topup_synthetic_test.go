package userapi

// 契约:合成客户(隔离空间负数 ID)拒绝充值——见 adopted note
// 2026-09-03-synthetic-customer-recharge-boundary.md。
// 真实客户正常调通,负数 cid 直接 42200 参数非法。

import (
	"net/http"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestPortal_TopupSyntheticReject(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)

	// 合成客户:负数 cid(对应 portal.NextSyntheticCustomerID 输出段)
	synthTok, _ := signCustomerToken(mgr, -1, "13800000000")
	r, _ := newActionsRouter(&woWithDispatch{}, &fakeBilling{}, nil, nil, nil, nil, nil, nil)

	t.Run("合成客户 POST /topups 42200", func(t *testing.T) {
		w := userPortalDo(r, http.MethodPost, "/api/user/v1/topups", `{"amount":50,"payMethod":"wechat"}`, synthTok)
		code, _ := userPortalCode(t, w)
		if code != 42200 {
			t.Fatalf("code=%d want 42200 body=%s", code, w.Body.String())
		}
	})
}

func TestPortal_TopupNegativeCIDReject(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	r, _ := newActionsRouter(&woWithDispatch{}, &fakeBilling{}, nil, nil, nil, nil, nil, nil)

	for _, cid := range []int64{-1, -100, -9999999999} {
		tok, _ := signCustomerToken(mgr, cid, "13800000000")
		w := userPortalDo(r, http.MethodPost, "/api/user/v1/topups", `{"amount":50,"payMethod":"wechat"}`, tok)
		code, _ := userPortalCode(t, w)
		if code != 42200 {
			t.Fatalf("cid=%d code=%d want 42200 body=%s", cid, code, w.Body.String())
		}
	}
}
