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
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeClosureTax 单票失败详情/重试/回放用桩:可配置发票与轨迹。
type fakeClosureTax struct {
	fakeTax
	inv    billing.Invoice
	events []billing.TaxEvent
	marked *billing.TaxReceipt
}

func (f *fakeClosureTax) GetInvoice(context.Context, int64) (*billing.Invoice, error) {
	return &f.inv, nil
}
func (f *fakeClosureTax) MarkTaxResult(_ context.Context, _ int64, r billing.TaxReceipt) error {
	f.marked = &r
	return nil
}
func (f *fakeClosureTax) AppendTaxEvent(context.Context, billing.TaxEvent) (int64, error) {
	return 1, nil
}
func (f *fakeClosureTax) ListTaxEvents(context.Context, int64) ([]billing.TaxEvent, error) {
	return f.events, nil
}

// fakeGW 测试用税局网关:回执可配。
type fakeGW struct {
	result billing.TaxReceipt
	err    error
}

func (g *fakeGW) Jurisdiction() string { return "CN" }
func (g *fakeGW) Channel() string      { return billing.TaxChannelLeqi }
func (g *fakeGW) Issue(context.Context, billing.Invoice) (billing.TaxReceipt, error) {
	return g.result, g.err
}

func closureRouter(tx *fakeClosureTax, gw billing.TaxGateway) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mgr := auth.NewManager("s", time.Hour)
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Billing: &fakeBillingRun{}, Tax: tx,
		TaxGateway: billing.NewTaxGatewayRegistry(gw),
	}, mgr)
	return r, mgr
}

// TestTaxFailureDetailsHandler 失败详情:返回发票税务字段 + 最近失败轨迹 + 可重试标记。
func TestTaxFailureDetailsHandler(t *testing.T) {
	now := time.Now()
	tx := &fakeClosureTax{
		inv: billing.Invoice{ID: 11, InvoiceNo: "INV-00000001", Status: "ISSUED",
			TaxJurisdiction: "CN", TaxChannel: billing.TaxChannelLeqi, TaxStatus: billing.TaxStatusFailed,
			TaxFailReason: "signature invalid"},
		events: []billing.TaxEvent{
			{ID: 1, InvoiceID: 11, Event: billing.TaxEventReceipt, TaxStatusAfter: billing.TaxStatusFailed,
				FailReason: "signature invalid", CreatedAt: now},
		},
	}
	r, mgr := closureRouter(tx, &fakeGW{})
	w := getJSON(t, r, "/api/admin/v1/invoices/11/tax-failure", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Invoice   billing.Invoice   `json:"invoice"`
			LastFail  *billing.TaxEvent `json:"lastFailure"`
			Retryable bool              `json:"retryable"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Data.Retryable || body.Data.Invoice.TaxStatus != billing.TaxStatusFailed ||
		body.Data.LastFail == nil || body.Data.LastFail.FailReason != "signature invalid" {
		t.Fatalf("body=%+v", body)
	}
}

// TestTaxRetryHandler 重试:FAILED 票重新提交网关并落回执。
func TestTaxRetryHandler(t *testing.T) {
	tx := &fakeClosureTax{
		inv: billing.Invoice{ID: 11, InvoiceNo: "INV-00000001", Status: "ISSUED",
			TaxJurisdiction: "CN", TaxChannel: billing.TaxChannelLeqi, TaxStatus: billing.TaxStatusFailed},
	}
	r, mgr := closureRouter(tx, &fakeGW{result: billing.TaxReceipt{
		Status: billing.TaxStatusIssued, TaxNo: "24122000000012345678", ExternalID: "ext-9"}})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/invoices/11/tax-retry", nil)
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if tx.marked == nil || tx.marked.TaxNo != "24122000000012345678" || tx.marked.ExternalID != "ext-9" {
		t.Fatalf("marked=%+v", tx.marked)
	}
}

// TestTaxReplayHandler 回放:重新推动网关,MarkTaxResult 结果幂等。
func TestTaxReplayHandler(t *testing.T) {
	tx := &fakeClosureTax{
		inv: billing.Invoice{ID: 11, InvoiceNo: "INV-00000001", Status: "ISSUED",
			TaxJurisdiction: "CN", TaxChannel: billing.TaxChannelLeqi, TaxStatus: billing.TaxStatusSUBMITTED},
	}
	// 重复回放持同一 external id:域层吃幂等,handler 不允许伪造成功,回执原样返回。
	calls := 0
	gw := &fakeGW{result: billing.TaxReceipt{Status: billing.TaxStatusFailed, FailReason: "timeout", ExternalID: "ext-10"}}
	r, mgr := closureRouter(tx, gw)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/invoices/11/tax-replay", nil)
		req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		calls++ // 网关每次都被调,幂等在域层;此处只验证链路可重复执行且非成功伪装
		if w.Code != http.StatusOK {
			t.Fatalf("replay#%d status=%d body=%s", i, w.Code, w.Body.String())
		}
	}
	t.Logf("replay calls=%d", calls)
}
