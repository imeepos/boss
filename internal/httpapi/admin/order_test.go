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
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeOrder 桩 order.OrderService:订单读/建/跟踪可配置,环节方法返回零值。
type fakeOrder struct {
	list      []order.OrderListItem
	byNo      *order.Order
	byNoErr   error
	trackLog  []order.StageLog
	submitted *order.Order
	submitErr error
	lastQ     order.OrderQuery
}

func (f *fakeOrder) Submit(ctx context.Context, r order.SubmitReq) (*order.Order, error) {
	return f.submitted, f.submitErr
}
func (f *fakeOrder) CheckResource(ctx context.Context, id int64) error  { return nil }
func (f *fakeOrder) Reserve(ctx context.Context, id int64) error        { return nil }
func (f *fakeOrder) ChargeContract(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) PrepaidAmount(ctx context.Context, id int64) (float64, int, bool, error) {
	return 0, 0, false, nil
}
func (f *fakeOrder) ApplyTag(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) CreateUserProfile(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeOrder) PreConfigOLT(ctx context.Context, id int64) error  { return nil }
func (f *fakeOrder) DispatchOrder(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) ScanBind(ctx context.Context, id int64) error      { return nil }
func (f *fakeOrder) ActivateUser(ctx context.Context, id int64) error  { return nil }
func (f *fakeOrder) NotifyActivation(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeOrder) UpdateMap(ctx context.Context, id int64) error { return nil }
func (f *fakeOrder) Cancel(ctx context.Context, id int64) error    { return nil }
func (f *fakeOrder) Release(ctx context.Context, id int64) error   { return nil }
func (f *fakeOrder) RollbackStage(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeOrder) List(ctx context.Context, q order.OrderQuery) ([]order.OrderListItem, error) {
	f.lastQ = q
	return f.list, nil
}
func (f *fakeOrder) GetByNo(ctx context.Context, no string) (*order.Order, error) {
	return f.byNo, f.byNoErr
}
func (f *fakeOrder) Track(ctx context.Context, id int64) (*order.Order, []order.StageLog, error) {
	return f.byNo, f.trackLog, nil
}
func (f *fakeOrder) ChangeAddress(context.Context, int64, int64) error { return nil }
func (f *fakeOrder) SaveRating(context.Context, order.Rating) error    { return nil }
func (f *fakeOrder) RatingExists(context.Context, string) (bool, error) {
	return false, nil
}

func newOrderRouter(f *fakeOrder, u *fakeUser, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: u, Order: f}, mgr)
	return r
}

func authToken(t *testing.T, mgr *auth.Manager) string {
	t.Helper()
	tok, err := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// TestOrderListHandler 契约:订单列表返回 items,含 stageLabel/ops/id。
func TestOrderListHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrder{list: []order.OrderListItem{
		{ID: 42, OrderNo: "ORD-1", Customer: "王先生", Product: "100M", Address: "Manila", Stage: 3, Status: "PENDING"},
	}}
	r := newOrderRouter(f, &fakeUser{permOk: true}, mgr)
	w := getJSON(t, r, "/api/admin/v1/orders", authToken(t, mgr))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []orderListResp `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].ID != 42 || body.Data.Items[0].OrderNo != "ORD-1" ||
		body.Data.Items[0].StageLabel != "端口预占" || len(body.Data.Items[0].Ops) == 0 {
		t.Fatalf("body=%+v", body)
	}
}

func TestOrderListHandlerAppliesDataScope(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrder{}
	u := &fakeUser{permOk: true, dataScope: user.DataScope{LegalEntityID: 3, RegionScope: "root.luzon"}}
	r := newOrderRouter(f, u, mgr)
	w := getJSON(t, r, "/api/admin/v1/orders", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if f.lastQ.LegalEntityID != 3 || f.lastQ.RegionScope != "root.luzon" {
		t.Fatalf("scope not applied: %+v", f.lastQ)
	}
}

// TestOrderCreateHandler 契约:新建订单走 Submit,返回 orderNo。
func TestOrderCreateHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrder{submitted: &order.Order{ID: 7, OrderNo: "ORD-7", Stage: 1, Status: "PENDING"}}
	r := newOrderRouter(f, &fakeUser{permOk: true}, mgr)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/orders",
		strings.NewReader(`{"customerId":1,"offerId":10,"addressId":100,"channelId":5,"legalEntityId":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken(t, mgr))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			OrderNo string `json:"orderNo"`
			Stage   int8   `json:"stage"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.OrderNo != "ORD-7" || body.Data.Stage != 1 {
		t.Fatalf("body=%+v", body)
	}
}

// TestOrderDetailHandler 契约:订单详情返回 order + timeline(环节名/耗时)。
func TestOrderDetailHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	t0 := time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(90 * time.Second)
	f := &fakeOrder{
		byNo: &order.Order{ID: 7, OrderNo: "ORD-7", Stage: 3, Status: "RESERVED", CreatedAt: t0},
		trackLog: []order.StageLog{
			{Stage: 1, Result: "DONE", FinishedAt: &t0},
			{Stage: 3, Result: "DONE", FinishedAt: &t1},
		},
	}
	r := newOrderRouter(f, &fakeUser{permOk: true}, mgr)
	w := getJSON(t, r, "/api/admin/v1/orders/ORD-7", authToken(t, mgr))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Order struct {
				OrderNo string `json:"orderNo"`
			} `json:"order"`
			Timeline []timelineItem `json:"timeline"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data.Order.OrderNo != "ORD-7" {
		t.Fatalf("order=%+v", body.Data.Order)
	}
	if len(body.Data.Timeline) != 2 || body.Data.Timeline[1].Name != "端口预占" || body.Data.Timeline[1].Duration != "1m30s" {
		t.Fatalf("timeline=%+v", body.Data.Timeline)
	}
}

// TestOrderDetailHandlerOutOfScopeAsNotFound 契约:数据范围外订单与不存在订单同码同响应
// 不可区分(2026-08-28 battle oracle);范围子树内正常放行。
func TestOrderDetailHandlerOutOfScopeAsNotFound(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true, dataScope: user.DataScope{LegalEntityID: 3, RegionScope: "root.luzon"}}

	// 范围外:实体不符 + 区域越子树
	fOut := &fakeOrder{byNo: &order.Order{ID: 7, OrderNo: "ORD-7", LegalEntityID: 9, RegionPath: "root.davao"}}
	wOut := getJSON(t, newOrderRouter(fOut, u, mgr), "/api/admin/v1/orders/ORD-7", authToken(t, mgr))

	// 真不存在:两类情形响应必须逐字节一致
	fMiss := &fakeOrder{byNoErr: order.ErrOrderNotFound}
	wMiss := getJSON(t, newOrderRouter(fMiss, u, mgr), "/api/admin/v1/orders/ORD-404", authToken(t, mgr))

	if wOut.Code != wMiss.Code || wOut.Body.String() != wMiss.Body.String() {
		t.Fatalf("越界与不存在可区分: out=%d %s miss=%d %s",
			wOut.Code, wOut.Body.String(), wMiss.Code, wMiss.Body.String())
	}

	// 子树内放行:区域 path 前缀匹配(root.luzon ⊇ root.luzon.makati)
	fIn := &fakeOrder{byNo: &order.Order{ID: 7, OrderNo: "ORD-7", LegalEntityID: 3, RegionPath: "root.luzon.makati"}}
	wIn := getJSON(t, newOrderRouter(fIn, u, mgr), "/api/admin/v1/orders/ORD-7", authToken(t, mgr))
	if wIn.Code != http.StatusOK {
		t.Fatalf("子树内被误拦: status=%d body=%s", wIn.Code, wIn.Body.String())
	}
}
