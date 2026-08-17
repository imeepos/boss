package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeBilling 桩 billing.BillingService。
type fakeBilling struct{ bills []billing.Bill }

func (f *fakeBilling) ListBills(context.Context, int64) ([]billing.Bill, error) { return f.bills, nil }
func (f *fakeBilling) CreateBill(context.Context, billing.Bill) (int64, error)  { return 0, nil }
func (f *fakeBilling) GetBill(context.Context, int64) (*billing.Bill, error)    { return nil, nil }
func (f *fakeBilling) ListPayments(context.Context, int64) ([]billing.Payment, error) {
	return nil, nil
}
func (f *fakeBilling) CreatePayment(context.Context, billing.Payment) (int64, error) { return 0, nil }
func (f *fakeBilling) GenerateBills(context.Context, string) (int, error)            { return 0, nil }

// fakeArrears 桩 billing.ArrearsService。
type fakeArrears struct {
	items    []billing.ArrearsItem
	appended *billing.StopResumeTask
}

func (f *fakeArrears) GetArrears(context.Context, int64) (*billing.Arrears, error) { return nil, nil }
func (f *fakeArrears) UpsertArrears(context.Context, billing.Arrears) (int64, error) {
	return 0, nil
}
func (f *fakeArrears) ListArrears(context.Context) ([]billing.ArrearsItem, error) {
	return f.items, nil
}
func (f *fakeArrears) ListStopResumeTasks(context.Context, int64) ([]billing.StopResumeTask, error) {
	return nil, nil
}
func (f *fakeArrears) AppendStopResumeTask(ctx context.Context, t billing.StopResumeTask) (int64, error) {
	f.appended = &t
	return 1, nil
}

// fakeAaa 桩 aaa.AaaService。
type fakeAaa struct{ lo *aaa.LoAccount }

func (f *fakeAaa) ListLoAccounts(context.Context) ([]aaa.LoAccount, error)       { return nil, nil }
func (f *fakeAaa) CreateLoAccount(context.Context, aaa.LoAccount) (int64, error) { return 0, nil }
func (f *fakeAaa) GetLoAccountByLoid(context.Context, string) (*aaa.LoAccount, error) {
	return f.lo, nil
}
func (f *fakeAaa) GetLoAccountByCustomer(context.Context, int64) (*aaa.LoAccount, error) {
	return f.lo, nil
}
func (f *fakeAaa) AppendCdr(context.Context, aaa.CdrRecord) (int64, error)     { return 0, nil }
func (f *fakeAaa) ListCdrs(context.Context, string) ([]aaa.CdrRecord, error)   { return nil, nil }
func (f *fakeAaa) AppendAuthLog(context.Context, aaa.AuthLog) (int64, error)   { return 0, nil }
func (f *fakeAaa) ListAuthLogs(context.Context, string) ([]aaa.AuthLog, error) { return nil, nil }
func (f *fakeAaa) SuspendLoAccount(context.Context, int64) error               { return nil }
func (f *fakeAaa) ResumeLoAccount(context.Context, int64) error                { return nil }

func newBillingRouter(b *fakeBilling, ar *fakeArrears, aa *fakeAaa, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, &Application{
		User: &fakeUser{permOk: true}, Billing: b, Arrears: ar, Aaa: aa,
	}, mgr)
	return r
}

// TestBillingListHandler 契约:欠费列表可查。
func TestBillingListHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ar := &fakeArrears{items: []billing.ArrearsItem{
		{CustomerID: 1, CustomerName: "王先生", Amount: 299.00, Days: 30, Status: "催收中"},
	}}
	r := newBillingRouter(&fakeBilling{}, ar, &fakeAaa{}, mgr)
	w := getJSON(t, r, "/api/v1/arrears", authToken(t, mgr))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []billing.ArrearsItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].CustomerName != "王先生" {
		t.Fatalf("body=%+v", body)
	}
}

// TestStopResumeHandler 契约:停机为客户 1:1 LO 账号生成 STOP 任务。
func TestStopResumeHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ar := &fakeArrears{}
	aa := &fakeAaa{lo: &aaa.LoAccount{ID: 88, CustomerID: 9, Loid: "LOID-9"}}
	r := newBillingRouter(&fakeBilling{}, ar, aa, mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/arrears/9/stop", nil)
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if ar.appended == nil || ar.appended.Action != "STOP" || ar.appended.LoAccountID != 88 || ar.appended.CustomerID != 9 {
		t.Fatalf("appended=%+v", ar.appended)
	}
}
