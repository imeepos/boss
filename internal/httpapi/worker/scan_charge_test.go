// 师傅端现场收款 handler 测试:预取应收(预付费同环节4 口径)、payMethods 随 stripe 配置、
// 落账 payments(pay_no + method 映射)、非法方式校验。
package workerapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// fakeChargeOrder 桩 order.OrderService:Track 带客户归属,PrepaidAmount 可配。
type fakeChargeOrder struct {
	order.OrderService
	customerID int64
	amount     float64
	months     int
	prepaid    bool
}

func (f *fakeChargeOrder) Track(context.Context, int64) (*order.Order, []order.StageLog, error) {
	return &order.Order{ID: 1, OrderNo: "ORD-1", CustomerID: f.customerID, Stage: 9, Status: "INSTALLING"}, nil, nil
}

func (f *fakeChargeOrder) PrepaidAmount(context.Context, int64) (float64, int, bool, error) {
	return f.amount, f.months, f.prepaid, nil
}

// fakeChargeBilling 桩 billing.BillingService:仅捕获 CreatePayment 入参。
type fakeChargeBilling struct {
	billing.BillingService
	payments []billing.Payment
}

func (f *fakeChargeBilling) CreatePayment(_ context.Context, p billing.Payment) (int64, error) {
	f.payments = append(f.payments, p)
	return int64(len(f.payments)), nil
}

// chargeTestRouterWith 现场收款路由装配;stripe 为 nil 或已配置 Dynamic 控制 payMethods。
func chargeTestRouterWith(t *testing.T, fo *fakeChargeOrder, fb *fakeChargeBilling, dyn *stripe.Dynamic) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	r := gin.New()
	a := &app.Application{
		Worker: &fakePortalWorkerSvc{},
		WorkOrder: &fakePortalWorkOrder{tickets: []order.DispatchTicket{
			{TicketID: 1, TicketNo: "DT-1", OrderID: 1, WorkerID: 7, Status: "DOING"},
		}},
		Order:      fo,
		Billing:    fb,
		Portal:     portal.NewMemory(),
		Stripe:     dyn,
		Automation: app.NewAutomation(fo, nil),
	}
	Register(r, a, newWorkerJWTManager())
	return r
}

// chargeGet 预取结果 map(健壮取数)。
func chargeGet(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	if asNum(out["_status"]) != 200 {
		t.Fatalf("get charge status: %v", out)
	}
	data, ok := out["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data: %v", out)
	}
	return data
}

// asNum 任意 JSON 数值 → float64(兼容 int / float64 / json.Number)。
func asNum(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	}
	return 0
}

// asStr 任意值 → 字符串。
func asStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}

// TestWorkerChargeGet_Prepaid 预付费订单:应收=月数x月费,描述预缴 N 个月;未配 stripe 仅线下。
func TestWorkerChargeGet_Prepaid(t *testing.T) {
	tok := portalGrabToken(t)
	fo := &fakeChargeOrder{customerID: 3, amount: 199, months: 6, prepaid: true}
	fb := &fakeChargeBilling{}
	r := chargeTestRouterWith(t, fo, fb, nil) // stripe 未配置

	data := chargeGet(t, portalWorkerDo(r, "GET", "/api/worker/v1/tickets/DT-1/charge", "", tok))
	if asNum(data["amountDue"]) != 199 {
		t.Fatalf("amountDue want 199, got %v", data["amountDue"])
	}
	methods := data["payMethods"].([]any)
	if len(methods) != 3 || methods[0] != "CASH" || methods[1] != "QR" || methods[2] != "POS" {
		t.Fatalf("未配 stripe 应仅线下 CASH/QR/POS, got %v", methods)
	}
	if asStr(data["amountDesc"]) != "预缴 6 个月" {
		t.Fatalf("amountDesc want 预缴 6 个月, got %v", data["amountDesc"])
	}
}

// TestWorkerChargeGet_StripeReady 已配 stripe:payMethods 含线上卡收款 CARD。
func TestWorkerChargeGet_StripeReady(t *testing.T) {
	tok := portalGrabToken(t)
	fo := &fakeChargeOrder{customerID: 3, amount: 99.5, months: 1, prepaid: true}
	dyn := stripe.NewDynamic(func(context.Context) (stripe.Config, error) {
		return stripe.Config{Enabled: true, APIKey: "sk_test_x"}, nil
	})
	r := chargeTestRouterWith(t, fo, &fakeChargeBilling{}, dyn)

	data := chargeGet(t, portalWorkerDo(r, "GET", "/api/worker/v1/tickets/DT-1/charge", "", tok))
	methods := data["payMethods"].([]any)
	if len(methods) != 4 || methods[0] != "CARD" {
		t.Fatalf("已配 stripe 应含 CARD 且居首, got %v", methods)
	}
}

// TestWorkerChargeGet_Postpaid 后付费订单:无强制应收(amountDue=0),提示实收自填。
func TestWorkerChargeGet_Postpaid(t *testing.T) {
	tok := portalGrabToken(t)
	fo := &fakeChargeOrder{customerID: 3, prepaid: false}
	r := chargeTestRouterWith(t, fo, &fakeChargeBilling{}, nil)

	data := chargeGet(t, portalWorkerDo(r, "GET", "/api/worker/v1/tickets/DT-1/charge", "", tok))
	if asNum(data["amountDue"]) != 0 {
		t.Fatalf("后付费 amountDue want 0, got %v", data["amountDue"])
	}
	if asStr(data["amountDesc"]) != postpaidChargeDesc {
		t.Fatalf("amountDesc want 后付费自填, got %v", data["amountDesc"])
	}
}

// TestWorkerChargePost_Records 线下收款落账:pay_no 派单 + method=offline + 客户归属。
func TestWorkerChargePost_Records(t *testing.T) {
	tok := portalGrabToken(t)
	fo := &fakeChargeOrder{customerID: 3, amount: 199, months: 6, prepaid: true}
	fb := &fakeChargeBilling{}
	r := chargeTestRouterWith(t, fo, fb, nil)

	out := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/charge",
		`{"amount":199,"payMethod":"CASH"}`, tok)
	if asNum(out["_status"]) != 200 {
		t.Fatalf("post charge status: %v", out)
	}
	if len(fb.payments) != 1 {
		t.Fatalf("want 1 payment recorded, got %d", len(fb.payments))
	}
	p := fb.payments[0]
	if p.Method != "offline" || p.CustomerID != 3 || p.Amount != 199 || p.Status != "SUCCESS" {
		t.Fatalf("payment mismatch: %+v", p)
	}
	if p.PayNo == "" {
		t.Fatal("payNo should be assigned")
	}
	data := out["data"].(map[string]any)
	if data["payNo"] != p.PayNo {
		t.Fatalf("response payNo %v != recorded %v", data["payNo"], p.PayNo)
	}
}

// TestWorkerChargePost_CardMethod CARD 落账 method=card(线上)。
func TestWorkerChargePost_CardMethod(t *testing.T) {
	tok := portalGrabToken(t)
	fo := &fakeChargeOrder{customerID: 3, amount: 199, months: 6, prepaid: true}
	fb := &fakeChargeBilling{}
	r := chargeTestRouterWith(t, fo, fb, nil)

	out := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/charge",
		`{"amount":199,"payMethod":"CARD"}`, tok)
	if asNum(out["_status"]) != 200 {
		t.Fatalf("post charge status: %v", out)
	}
	if fb.payments[0].Method != "card" {
		t.Fatalf("CARD payment method want card, got %s", fb.payments[0].Method)
	}
}

// TestWorkerChargePost_InvalidMethod 未知方式拒绝(biz 42200),不落账。
func TestWorkerChargePost_InvalidMethod(t *testing.T) {
	tok := portalGrabToken(t)
	fo := &fakeChargeOrder{customerID: 3, amount: 199, prepaid: true}
	fb := &fakeChargeBilling{}
	r := chargeTestRouterWith(t, fo, fb, nil)

	out := portalWorkerDo(r, "POST", "/api/worker/v1/tickets/DT-1/charge",
		`{"amount":199,"payMethod":"BITCOIN"}`, tok)
	if asNum(out["code"]) != 42200 {
		t.Fatalf("invalid payMethod want code 42200, got %v", out)
	}
	if len(fb.payments) != 0 {
		t.Fatalf("invalid method should not record, got %+v", fb.payments)
	}
}
