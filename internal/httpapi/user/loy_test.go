package userapi

// 积分域端点冒烟:余额/流水 + 兑换。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakeLoy struct {
	balance int64
	entries []loy.Entry
	coupon  string
	cost    int64
}

func (f *fakeLoy) Balance(context.Context, int64) (int64, error) { return f.balance, nil }
func (f *fakeLoy) Entries(context.Context, int64) ([]loy.Entry, error) {
	return f.entries, nil
}
func (f *fakeLoy) Adjust(_ context.Context, _ int64, delta int64, _ string) (int64, error) {
	f.balance += delta
	return f.balance, nil
}
func (f *fakeLoy) Exchange(context.Context, int64, int64) (string, int64, error) {
	return f.coupon, f.cost, nil
}

func TestPortalPoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	cust := userPortalCust()
	fl := &fakeLoy{balance: 500, coupon: "CPN-loy", cost: 300,
		entries: []loy.Entry{{EntryID: 1, Delta: 500, BalanceAfter: 500, Reason: "ADMIN_ADJUST"}}}
	r := gin.New()
	Register(r, &app.Application{
		Customer: &userPortalCustSvc{c: cust},
		Portal:   portal.NewMemory(),
		Points:   fl,
	}, mgr)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/points", ``, tok)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Balance int64 `json:"balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Code != 0 || resp.Data.Balance != 500 {
		t.Fatalf("points resp=%s", w.Body.String())
	}

	w = userPortalDo(r, http.MethodPost, "/api/user/v1/points/exchange", `{"templateId":5}`, tok)
	var ex struct {
		Data struct {
			CouponID string `json:"couponId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ex); err != nil || ex.Data.CouponID != "CPN-loy" {
		t.Fatalf("exchange resp=%s", w.Body.String())
	}
}
