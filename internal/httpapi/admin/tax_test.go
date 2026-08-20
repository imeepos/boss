package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeTax 桩 billing.TaxService。
type fakeTax struct {
	invoices  []billing.Invoice
	runPeriod string
	runResult billing.InvoiceRunResult
	voided    *struct {
		id     int64
		reason string
	}
	reissued int64
}

func (f *fakeTax) ListInvoices(context.Context, int64) ([]billing.Invoice, error) {
	return f.invoices, nil
}
func (f *fakeTax) IssueInvoicesForPeriod(_ context.Context, period string) (billing.InvoiceRunResult, error) {
	f.runPeriod = period
	return f.runResult, nil
}
func (f *fakeTax) IssueInvoiceForBill(context.Context, int64, string) (*billing.Invoice, error) {
	return nil, billing.ErrNotFound
}
func (f *fakeTax) VoidInvoice(_ context.Context, id int64, reason string) error {
	f.voided = &struct {
		id     int64
		reason string
	}{id, reason}
	return nil
}
func (f *fakeTax) ReissueInvoice(_ context.Context, id int64) (*billing.Invoice, error) {
	f.reissued = id
	return &billing.Invoice{ID: 12, InvoiceNo: "INV-00000002"}, nil
}
func (f *fakeTax) GetInvoice(_ context.Context, _ int64) (*billing.Invoice, error) {
	return &billing.Invoice{ID: 11, InvoiceNo: "INV-00000001", Status: "ISSUED",
		TaxJurisdiction: "CN", TaxChannel: "manual", TaxStatus: "PENDING"}, nil
}
func (f *fakeTax) BackfillTaxNo(context.Context, int64, string) error { return nil }
func (f *fakeTax) MarkTaxResult(context.Context, int64, billing.TaxReceipt) error {
	return nil
}

// fakeBillingRun 记录出账调用的 fakeBilling。
type fakeBillingRun struct {
	fakeBilling
	period    string
	generated int
}

func (f *fakeBillingRun) GenerateBills(_ context.Context, period string) (int, error) {
	f.period = period
	return f.generated, nil
}

func newTaxRouter(b *fakeBillingRun, tx *fakeTax, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Billing: b, Tax: tx,
	}, mgr)
	return r
}

// TestBillingRunHandler 契约(CT-007):出账完成即自动开票,同端点幂等可重跑。
func TestBillingRunHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	b := &fakeBillingRun{generated: 3}
	tx := &fakeTax{runResult: billing.InvoiceRunResult{Issued: 3, FailedIDs: []int64{}}}
	r := newTaxRouter(b, tx, mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/billing-runs",
		strings.NewReader(`{"period":"2026-08"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if b.period != "2026-08" || tx.runPeriod != "2026-08" {
		t.Fatalf("period b=%q tx=%q", b.period, tx.runPeriod)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Bills    int                      `json:"bills"`
			Invoices billing.InvoiceRunResult `json:"invoices"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Bills != 3 || body.Data.Invoices.Issued != 3 {
		t.Fatalf("body=%+v", body)
	}
}

// TestInvoiceListAndVoidHandler 契约:发票可查、可作废(编号保留由域层保证)。
func TestInvoiceListAndVoidHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	tx := &fakeTax{invoices: []billing.Invoice{
		{ID: 11, InvoiceNo: "INV-00000001", BillNo: "BILL-202608-201",
			NetAmount: 999.0, VatAmount: 119.88, TotalAmount: 1118.88, Status: "ISSUED"},
	}}
	r := newTaxRouter(&fakeBillingRun{}, tx, mgr)

	w := getJSON(t, r, "/api/admin/v1/invoices", authToken(t, mgr))
	var list struct {
		Code int `json:"code"`
		Data struct {
			Items []billing.Invoice `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Code != 0 || len(list.Data.Items) != 1 || list.Data.Items[0].InvoiceNo != "INV-00000001" {
		t.Fatalf("list=%+v", list)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/invoices/11/void",
		strings.NewReader(`{"reason":"客户要求重开"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("void status=%d body=%s", w.Code, w.Body.String())
	}
	if tx.voided == nil || tx.voided.id != 11 || tx.voided.reason != "客户要求重开" {
		t.Fatalf("voided=%+v", tx.voided)
	}
}

// TestRecordPaymentHandler 契约:收款落账经 POST /payments(流水+账单 PAID 同事务)。
func TestRecordPaymentHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	paid := &fakePaid{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User:    &fakeUser{permOk: true},
		Billing: paid,
		Tax:     &fakeTax{},
	}, mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/payments",
		strings.NewReader(`{"payNo":"PAY-20260818-001","billId":1,"amount":999,"method":"wechat"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if paid.recorded == nil || paid.recorded.PayNo != "PAY-20260818-001" || paid.recorded.BillID != 1 {
		t.Fatalf("recorded=%+v", paid.recorded)
	}
}

// fakePaid 记录 RecordPayment 入参的桩。
type fakePaid struct {
	fakeBilling
	recorded *billing.Payment
}

func (f *fakePaid) RecordPayment(_ context.Context, p billing.Payment) (int64, error) {
	f.recorded = &p
	return 1, nil
}

// TestTaxBackfillHandler 契约(人工通道):税局平台开具后回填票号。
func TestTaxBackfillHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Billing: &fakeBillingRun{}, Tax: &fakeTax{},
	}, mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/invoices/11/tax-backfill",
		strings.NewReader(`{"taxNo":"24122000000012345678"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// TestTaxSubmitHandler_NoGateway 契约:未配置属地网关(manual 通道)时提示走回填,不误报成功。
func TestTaxSubmitHandler_NoGateway(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Billing: &fakeBillingRun{}, Tax: &fakeTax{},
	}, mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/invoices/11/tax-submit", nil)
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code == 0 {
		t.Fatalf("manual 通道不应直接成功: %s", w.Body.String())
	}
}
