package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
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

func (f *fakeBilling) ListPaymentsByCustomer(context.Context, int64) ([]billing.Payment, error) {
	return nil, nil
}
func (f *fakeBilling) CreatePayment(context.Context, billing.Payment) (int64, error) { return 0, nil }
func (f *fakeBilling) PaymentExistsByPayNo(context.Context, string) (bool, error)    { return false, nil }
func (f *fakeBilling) RecordTopup(context.Context, billing.Payment) (int64, error)   { return 0, nil }
func (f *fakeBilling) RecordPayment(context.Context, billing.Payment) (int64, error) { return 0, nil }
func (f *fakeBilling) RecordPaymentWithCoupon(_ context.Context, p billing.Payment) (billing.PaymentReceipt, error) {
	return billing.PaymentReceipt{Amount: p.Amount}, nil
}
func (f *fakeBilling) GenerateBills(context.Context, string) (int, error) { return 0, nil }
func (f *fakeBilling) RefundPayment(_ context.Context, _ int64, _ string) (*billing.Payment, error) {
	return nil, nil
}

// fakeArrears 桩 billing.ArrearsService。
type fakeArrears struct {
	items    []billing.ArrearsItem
	appended *billing.StopResumeTask
	task     *billing.StopResumeTask // GetStopResumeTask 返回
	updated  *struct {               // UpdateStopResumeStatus 落账
		id     int64
		status string
	}
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

func (f *fakeArrears) GetStopResumeTask(context.Context, int64) (*billing.StopResumeTask, error) {
	return f.task, nil
}

func (f *fakeArrears) UpdateStopResumeStatus(_ context.Context, id int64, status string) error {
	f.updated = &struct {
		id     int64
		status string
	}{id, status}
	return nil
}

// fakeRecon 桩 billing.ReconService。
type fakeRecon struct {
	batches   []billing.ReconBatch
	byBatchNo map[string]*billing.ReconBatch
	items     map[int64][]billing.ReconItem
	stated    int64
	settled   string
}

func (f *fakeRecon) ListReconciliations(context.Context) ([]billing.ReconBatch, error) {
	return f.batches, nil
}
func (f *fakeRecon) AppendReconciliation(context.Context, billing.ReconBatch) (int64, error) {
	return 1, nil
}
func (f *fakeRecon) SettleReconciliation(_ context.Context, batchNo string) error {
	f.settled = batchNo
	return nil
}
func (f *fakeRecon) GetReconciliation(_ context.Context, batchNo string) (*billing.ReconBatch, error) {
	if b, ok := f.byBatchNo[batchNo]; ok {
		return b, nil
	}
	return nil, billing.ErrNotFound
}
func (f *fakeRecon) RecordChannelStatement(_ context.Context, batchID int64, _ []billing.ChannelStatementRow) error {
	f.stated = batchID
	return nil
}
func (f *fakeRecon) ListReconciliationItems(_ context.Context, batchID int64) ([]billing.ReconItem, error) {
	return f.items[batchID], nil
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
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Billing: b, Arrears: ar, Aaa: aa,
	}, mgr)
	return r
}

// postAuth 带鉴权令牌发起 POST。
func postAuth(t *testing.T, r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRetryStopResumeTask 契约:失败停复机任务重试,重放 LO 迁移并回写 DONE。
func TestRetryStopResumeTask(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ar := &fakeArrears{task: &billing.StopResumeTask{
		ID: 7, CustomerID: 9, LoAccountID: 88, Action: "STOP", Status: "FAILED",
	}}
	r := newBillingRouter(&fakeBilling{}, ar, &fakeAaa{}, mgr)

	w := postAuth(t, r, "/api/admin/v1/stop-resume-tasks/7/retry", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			OK bool `json:"ok"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || !body.Data.OK {
		t.Fatalf("body=%+v", body)
	}
	if ar.updated == nil || ar.updated.id != 7 || ar.updated.status != "DONE" {
		t.Fatalf("updated=%+v", ar.updated)
	}
}

// TestRetryStopResumeTask_NotFailed 契约:非 FAILED 任务不可重试。
func TestRetryStopResumeTask_NotFailed(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ar := &fakeArrears{task: &billing.StopResumeTask{ID: 7, Action: "STOP", Status: "DONE"}}
	r := newBillingRouter(&fakeBilling{}, ar, &fakeAaa{}, mgr)

	w := postAuth(t, r, "/api/admin/v1/stop-resume-tasks/7/retry", authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Code == 0 {
		t.Fatalf("应拒绝非 FAILED 任务重试: %s", w.Body.String())
	}
}

// TestReconciliationHandlers 契约:对账批次可查、差异挂起批次可平账。
func TestReconciliationHandlers(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	rc := &fakeRecon{batches: []billing.ReconBatch{
		{BatchNo: "PC-20250816-04", Channel: "支付宝", ChannelAmount: 862005, SystemAmount: 861905, Status: "DIFF_PENDING"},
	}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Recon: rc,
	}, mgr)

	w := getJSON(t, r, "/api/admin/v1/reconciliations", authToken(t, mgr))
	var list struct {
		Code int `json:"code"`
		Data struct {
			Items []billing.ReconBatch `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Code != 0 || len(list.Data.Items) != 1 || list.Data.Items[0].BatchNo != "PC-20250816-04" {
		t.Fatalf("list=%+v", list)
	}

	w = postAuth(t, r, "/api/admin/v1/reconciliations/PC-20250816-04/settle", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if rc.settled != "PC-20250816-04" {
		t.Fatalf("settled=%q", rc.settled)
	}
}

// TestReconciliationItemsHandlers 契约:批次行级明细可查(camelCase 字段)、渠道流水可按行录入比对。
func TestReconciliationItemsHandlers(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	rc := &fakeRecon{
		byBatchNo: map[string]*billing.ReconBatch{
			"PC-20250816-04": {ID: 4, BatchNo: "PC-20250816-04", Status: "DIFF_PENDING"},
		},
		items: map[int64][]billing.ReconItem{4: {
			{ID: 1, BatchID: 4, ChannelRef: "PAY-20250820-001", Amount: 299, DiffKind: billing.DiffAmountMismatch},
		}},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: true}, Recon: rc}, mgr)

	w := getJSON(t, r, "/api/admin/v1/reconciliations/PC-20250816-04/items", authToken(t, mgr))
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []billing.ReconItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 0 || len(body.Data.Items) != 1 || body.Data.Items[0].DiffKind != billing.DiffAmountMismatch {
		t.Fatalf("body=%s", w.Body.String())
	}

	w = postJSONAuth(t, r, "/api/admin/v1/reconciliations/PC-20250816-04/statement",
		`{"rows":[{"channelRef":"PAY-20250820-001","amount":299}]}`, authToken(t, mgr))
	if w.Code != http.StatusOK || rc.stated != 4 {
		t.Fatalf("status=%d stated=%d body=%s", w.Code, rc.stated, w.Body.String())
	}
}

// TestBillingListHandler 契约:欠费列表可查。
func TestBillingListHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	ar := &fakeArrears{items: []billing.ArrearsItem{
		{CustomerID: 1, CustomerName: "王先生", Amount: 299.00, Days: 30, Status: "催收中"},
	}}
	r := newBillingRouter(&fakeBilling{}, ar, &fakeAaa{}, mgr)
	w := getJSON(t, r, "/api/admin/v1/arrears", authToken(t, mgr))

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

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/arrears/9/stop", nil)
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
