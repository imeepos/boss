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
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakePromotion 嵌入 Service 接口,仅实现 exchange-offers 用到的读方法(其余走 nil)。
type fakePromotion struct {
	promotion.Service
	offers []promotion.Template
}

func (f *fakePromotion) ListExchangeOffers(context.Context) ([]promotion.Template, error) {
	return f.offers, nil
}

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

func (f *fakeLoy) ListLevels(context.Context) ([]loy.Level, error)       { return nil, nil }
func (f *fakeLoy) CreateLevel(context.Context, loy.Level) (int64, error) { return 1, nil }
func (f *fakeLoy) DisableLevel(context.Context, int64) error             { return nil }
func (f *fakeLoy) TierOf(context.Context, int64) (*loy.Level, error)     { return nil, nil }
func (f *fakeLoy) ListTasks(context.Context) ([]loy.Task, error)         { return nil, nil }
func (f *fakeLoy) CreateTask(context.Context, loy.Task) (int64, error)   { return 1, nil }
func (f *fakeLoy) DisableTask(context.Context, int64) error              { return nil }
func (f *fakeLoy) CompleteTask(_ context.Context, _ int64, _ int64) (int64, error) {
	return f.balance, nil
}
func (f *fakeLoy) TaskStatus(context.Context, int64) (map[int64]string, error) {
	return nil, nil
}
func (f *fakeLoy) EarnForPayment(context.Context, int64, int64, int64) (int64, error) {
	return 0, nil
}
func (f *fakeLoy) RollbackPayment(context.Context, int64, int64) (int64, error) {
	return 0, nil
}
func (f *fakeLoy) EarnRuleOf(context.Context) (*loy.EarnRule, error) { return nil, nil }
func (f *fakeLoy) SaveEarnRule(context.Context, loy.EarnRule) (int64, error) {
	return 1, nil
}
func (f *fakeLoy) ExpireDue(context.Context) (int, error) { return 0, nil }

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
		Promotion: &fakePromotion{offers: []promotion.Template{
			{TemplateID: 5, Name: "10元话费券", Type: "CASH", FaceValue: 1000, PointsPrice: 100},
			{TemplateID: 7, Name: "20元满减券", Type: "FULL_CUT", FaceValue: 2000, Threshold: 10000, PointsPrice: 180},
		}},
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

	w = userPortalDo(r, http.MethodGet, "/api/user/v1/points/exchange-offers", ``, tok)
	var offers struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				TemplateID  int64  `json:"templateId"`
				Name        string `json:"name"`
				PointsPrice int64  `json:"pointsPrice"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &offers); err != nil || offers.Code != 0 {
		t.Fatalf("exchange-offers resp=%s", w.Body.String())
	}
	if len(offers.Data.Items) != 2 || offers.Data.Items[0].TemplateID != 5 ||
		offers.Data.Items[0].Name != "10元话费券" || offers.Data.Items[0].PointsPrice != 100 {
		t.Fatalf("exchange-offers items unexpected: %s", w.Body.String())
	}
}
