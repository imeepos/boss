package userapi

// 报修催单/联系师傅、凭证与发票 PDF、自动缴费、profile plan 单测。

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// woWithDispatch 桩 WorkOrderService:在 userPortalWo 基础上补派单工单。
type woWithDispatch struct {
	userPortalWo
	dispatch []order.DispatchTicket
}

func (f *woWithDispatch) ListDispatchTickets(context.Context) ([]order.DispatchTicket, error) {
	return f.dispatch, nil
}

// fakeWorkerSvc 桩 worker.WorkerService:GetWorker 命中返回明文电话。
type fakeWorkerSvc struct{ byID map[int64]*worker.Worker }

func (f *fakeWorkerSvc) ListGroups(context.Context) ([]worker.Group, error) { return nil, nil }
func (f *fakeWorkerSvc) CreateGroup(context.Context, worker.Group) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerSvc) ListWorkers(context.Context, int64) ([]worker.Worker, error) {
	return nil, nil
}
func (f *fakeWorkerSvc) CreateWorker(context.Context, worker.Worker) (int64, error) {
	return 0, nil
}
func (f *fakeWorkerSvc) GetWorker(_ context.Context, id int64) (*worker.Worker, error) {
	return f.byID[id], nil
}

// billingWithPays 桩 Billing:ListPayments 返回可配置流水。
type billingWithPays struct {
	fakeBilling
	pays []billing.Payment
}

func (f *billingWithPays) ListPayments(context.Context, int64) ([]billing.Payment, error) {
	return f.pays, nil
}

// newActionsRouter 装配动作端点专用桩。
func newActionsRouter(wo *woWithDispatch, bill billing.BillingService, tax billing.TaxService,
	prods []customer.ProductOffer, list []order.OrderListItem, byNo *order.Order,
	ws worker.WorkerService, ud *fakeUserData) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	if ud == nil {
		ud = &fakeUserData{}
	}
	r := gin.New()
	Register(r, &app.Application{
		Customer: &userPortalCustSvc{c: userPortalCust()}, Billing: bill,
		Order: &fakeOrder{list: list, byNo: byNo}, Product: &fakeProduct{list: prods},
		WorkOrder: wo, Portal: portal.NewMemory(), UserData: ud,
		Tax: tax, Worker: ws,
	}, mgr)
	return r, mgr
}

func actionCtx() (*customer.Customer, []order.DispatchTicket) {
	cust := userPortalCust()
	d := []order.DispatchTicket{{
		TicketID: 3, TicketNo: "DSP-3", OrderID: 9, WorkerID: 55, WorkerName: "李师傅", Status: "DOING",
	}}
	return cust, d
}

// TestPortal_FaultUrge 催单:本人工单 ok+urgeNo;他人工单 404。
func TestPortal_FaultUrge(t *testing.T) {
	cust, _ := actionCtx()
	wo := &woWithDispatch{userPortalWo: userPortalWo{complts: []order.Complaint{
		{TicketNo: "TKT-1", CustomerID: cust.ID, Status: "OPEN"},
		{TicketNo: "TKT-2", CustomerID: 999, Status: "OPEN"},
	}}}
	r, mgr := newActionsRouter(wo, &fakeBilling{}, nil, nil, nil, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/faults/TKT-1/urge", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) || data["urgeNo"] == nil {
		t.Fatalf("urge resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/faults/TKT-2/urge", ``, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeNotFound) {
		t.Fatalf("urge foreign ticket resp=%s", w.Body.String())
	}
}

// TestPortal_FaultContact 联系师傅:已派单返回明文电话;未派单 404。
func TestPortal_FaultContact(t *testing.T) {
	cust, _ := actionCtx()
	wo := &woWithDispatch{userPortalWo: userPortalWo{complts: []order.Complaint{
		{TicketNo: "TKT-1", CustomerID: cust.ID, Status: "OPEN"},
	}}}
	_, wo.dispatch = actionCtx()
	ws := &fakeWorkerSvc{byID: map[int64]*worker.Worker{
		55: {ID: 55, Name: "李师傅", Phone: "13900005555"},
	}}
	list := []order.OrderListItem{{OrderNo: "ORD-1", Status: "INSTALLING"}}
	byNo := &order.Order{ID: 9, OrderNo: "ORD-1", CustomerID: cust.ID}
	r, mgr := newActionsRouter(wo, &fakeBilling{}, nil, nil, list, byNo, ws, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/faults/TKT-1/contact", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) || data["technicianPhone"] != "13900005555" {
		t.Fatalf("contact resp=%s", w.Body.String())
	}
	wo.dispatch = nil
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/faults/TKT-1/contact", ``, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeNotFound) {
		t.Fatalf("contact unassigned resp=%s", w.Body.String())
	}
}

// TestPortal_ReceiptAndInvoicePdf 凭证/发票 PDF:魔数 %PDF + content-type;未命中 404。
func TestPortal_ReceiptAndInvoicePdf(t *testing.T) {
	cust, _ := actionCtx()
	bill := &billingWithPays{pays: []billing.Payment{{PayNo: "PAY-1", BillID: 1, Amount: 99, Method: "wechat", Status: "SUCCESS"}}}
	bill.bills = []billing.Bill{{BillID: 1, BillNo: "B-1", CustomerID: cust.ID, Period: "2026-08", Amount: 99, Status: "UNPAID"}}
	tax := &fakeTaxStub{invs: []billing.Invoice{{
		InvoiceNo: "INV-00000001", BillNo: "B-1", CustomerID: cust.ID, Title: cust.Name,
		NetAmount: 88.39, VatRate: 0.12, VatAmount: 10.61, TotalAmount: 99, Status: "ISSUED",
	}}}
	r, mgr := newActionsRouter(&woWithDispatch{}, bill, tax, nil, nil, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/payments/PAY-1/receipt.pdf", ``, tok)
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "%PDF") ||
		!strings.Contains(w.Header().Get("Content-Type"), "application/pdf") {
		t.Fatalf("receipt.pdf code=%d type=%s", w.Code, w.Header().Get("Content-Type"))
	}
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/invoices/INV-00000001/pdf", ``, tok)
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "%PDF") {
		t.Fatalf("invoice.pdf code=%d", w.Code)
	}
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/payments/PAY-9/receipt.pdf", ``, tok)
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeNotFound) {
		t.Fatalf("missing receipt resp=%s", w.Body.String())
	}
}

// TestPortal_SmsLogin 短信验证码登录:App 发 {phone,mode:"sms",smsCode}。
func TestPortal_SmsLogin(t *testing.T) {
	r, _ := newActionsRouter(&woWithDispatch{}, &fakeBilling{}, nil, nil, nil, nil, nil, nil)
	_ = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/sms-code",
		`{"phone":"13900005678","scene":"register"}`, "")
	_ = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/register",
		`{"phone":"13900005678","smsCode":"123456","password":"password-10x"}`, "")
	_ = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/sms-code",
		`{"phone":"13900005678","scene":"login"}`, "")

	w := userPortalDo(r, http.MethodPost, "/api/user/v1/auth/login",
		`{"phone":"13900005678","mode":"sms","smsCode":"123456"}`, "")
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) || data["token"] == nil {
		t.Fatalf("sms login resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/auth/login",
		`{"phone":"13900005678","mode":"sms","smsCode":"000000"}`, "")
	if code, _ := userPortalCode(t, w); code != int(apitypes.CodeUnauthorized) {
		t.Fatalf("bad sms code resp=%s", w.Body.String())
	}
}

// TestPortal_AutoPay 自动缴费:GET 缺省关 → POST 开通 → GET 已开。
func TestPortal_AutoPay(t *testing.T) {
	cust, _ := actionCtx()
	r, mgr := newActionsRouter(&woWithDispatch{}, &fakeBilling{}, nil, nil, nil, nil, nil, nil)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/billing/auto-pay", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) || data["autoPayEnabled"] != false {
		t.Fatalf("autopay get resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodPost, "/api/user/v1/billing/auto-pay", `{"enabled":true}`, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) || data["autoPayEnabled"] != true {
		t.Fatalf("autopay set resp=%s", w.Body.String())
	}
	w = userPortalDo(r, http.MethodGet, "/api/user/v1/billing/auto-pay", ``, tok)
	if code, data := userPortalCode(t, w); code != int(apitypes.CodeOK) || data["autoPayEnabled"] != true {
		t.Fatalf("autopay re-get resp=%s", w.Body.String())
	}
}

// TestPortal_ProfilePlan /profile.plan:套餐名/月费 + 本月账单(最近未缴)。
func TestPortal_ProfilePlan(t *testing.T) {
	cust, _ := actionCtx()
	bills := []billing.Bill{
		{BillID: 1, BillNo: "B-1", CustomerID: cust.ID, Period: "2026-07", Amount: 99, Status: "PAID"},
		{BillID: 2, BillNo: "B-2", CustomerID: cust.ID, Period: "2026-08", Amount: 99, Status: "UNPAID"},
	}
	prods := []customer.ProductOffer{{ID: 101, Name: "家庭宽带100M", MonthlyFee: 99, Status: "PUBLISHED"}}
	ud := &fakeUserData{plans: []map[string]any{
		{"customerId": cust.ID, "productId": float64(101), "planName": "家庭宽带100M", "status": "ACTIVE"},
	}}
	r, mgr := newActionsRouter(&woWithDispatch{}, &fakeBilling{bills: bills}, nil, prods, nil, nil, nil, ud)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/profile", ``, tok)
	_, data := userPortalCode(t, w)
	plan, _ := data["plan"].(map[string]any)
	if plan == nil || plan["name"] != "家庭宽带100M" || plan["monthlyFee"] != float64(99) ||
		plan["currentBillDue"] != "2026-08" || plan["currentBillAmount"] != float64(99) {
		t.Fatalf("profile plan=%v", data["plan"])
	}
}
