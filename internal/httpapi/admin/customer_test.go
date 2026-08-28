package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakeCustomer struct {
	list  []customer.Customer
	lastQ customer.CustomerQuery
}

func (f *fakeCustomer) Create(context.Context, customer.Customer) (int64, error) { return 0, nil }
func (f *fakeCustomer) Get(context.Context, int64) (*customer.Customer, error)   { return nil, nil }
func (f *fakeCustomer) GetInScope(context.Context, int64, int64, string) (*customer.Customer, error) {
	return nil, nil
}
func (f *fakeCustomer) List(_ context.Context, q customer.CustomerQuery) ([]customer.Customer, error) {
	f.lastQ = q
	return f.list, nil
}

type fakeProduct struct {
	list     []customer.ProductOffer
	changed  *customer.ProductOffer // 记录最近一次调价入参
	updated  *customer.ProductOffer // 记录最近一次编辑入参
	statusTo string                 // 记录最近一次状态入参
}

func (f *fakeProduct) ListProducts(context.Context, int64) ([]customer.ProductOffer, error) {
	return f.list, nil
}
func (f *fakeProduct) CreateProduct(context.Context, customer.ProductOffer) (int64, error) {
	return 0, nil
}
func (f *fakeProduct) ListRegionOffers(context.Context, int64) ([]customer.RegionOffer, error) {
	return nil, nil
}
func (f *fakeProduct) CreateRegionOffer(context.Context, customer.RegionOffer) (int64, error) {
	return 0, nil
}
func (f *fakeProduct) ChangeProductPrice(_ context.Context, offerID int64, newFee float64, _ time.Time, reason string, _ int64) (int64, error) {
	f.changed = &customer.ProductOffer{ID: offerID, MonthlyFee: newFee}
	return 11, nil
}
func (f *fakeProduct) UpdateProduct(_ context.Context, offerID int64, name, bandwidth, category string) error {
	f.updated = &customer.ProductOffer{ID: offerID, Name: name, Bandwidth: bandwidth, Category: category}
	return nil
}
func (f *fakeProduct) UpdateProductStatus(_ context.Context, offerID int64, status string) error {
	f.statusTo = status
	return nil
}

type fakeRealName struct{}

func (f *fakeRealName) ListVerifications(context.Context, int64) ([]customer.RealNameVerification, error) {
	return nil, nil
}
func (f *fakeRealName) AppendVerification(context.Context, customer.RealNameVerification) (int64, error) {
	return 0, nil
}
func (f *fakeRealName) SubmitRealName(context.Context, customer.CustomerRealNameVerification) (int64, error) {
	return 0, nil
}
func (f *fakeRealName) GetLatest(context.Context, int64) (*customer.CustomerRealNameVerification, error) {
	return nil, customer.ErrRealNameNotFound
}
func (f *fakeRealName) Verify(context.Context, int64, string, string, string, int64) error {
	return nil
}

type fakeLedger struct{}

func (f *fakeLedger) ListCustomerHistories(context.Context, int64) ([]customer.CustomerHistory, error) {
	return nil, nil
}
func (f *fakeLedger) AppendCustomerHistory(context.Context, customer.CustomerHistory) (int64, error) {
	return 0, nil
}
func (f *fakeLedger) ListProductPriceHistories(context.Context, int64) ([]customer.ProductPriceHistory, error) {
	return nil, nil
}
func (f *fakeLedger) AppendProductPriceHistory(context.Context, customer.ProductPriceHistory) (int64, error) {
	return 0, nil
}
func (f *fakeLedger) ListRegionPriceHistories(context.Context, int64) ([]customer.RegionPriceHistory, error) {
	return nil, nil
}
func (f *fakeLedger) AppendRegionPriceHistory(context.Context, customer.RegionPriceHistory) (int64, error) {
	return 0, nil
}

func newCustomerRouter(c *fakeCustomer, p *fakeProduct, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: &fakeUser{permOk: true}, Customer: c, Product: p, RealName: &fakeRealName{}, CustomerLedger: &fakeLedger{},
	}, mgr)
	return r
}

// TestCustomerListHandler 契约:客户档案列表返回 items。
func TestCustomerListHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	c := &fakeCustomer{list: []customer.Customer{
		{ID: 1, Name: "王先生", Phone: "13800001111", RealNameStatus: "VERIFIED", ServiceStatus: "ACTIVE"},
	}}
	r := newCustomerRouter(c, &fakeProduct{}, mgr)
	w := getJSON(t, r, "/api/admin/v1/customers", authToken(t, mgr))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []customer.Customer `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].Name != "王先生" || body.Data.Items[0].RealNameStatus != "VERIFIED" {
		t.Fatalf("body=%+v", body)
	}
	if c.lastQ.LegalEntityID != 0 || c.lastQ.RegionScope != "" {
		t.Fatalf("scope=%+v", c.lastQ)
	}
}

func TestCustomerListHandlerAppliesDataScope(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	c := &fakeCustomer{}
	r := newCustomerRouter(c, &fakeProduct{}, mgr)
	w := getJSON(t, r, "/api/admin/v1/customers", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if c.lastQ.LegalEntityID != 0 || c.lastQ.RegionScope != "" {
		t.Fatalf("unexpected default scope=%+v", c.lastQ)
	}
}

func TestCustomerListHandlerAppliesDataScopeRestricted(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	fc := &fakeCustomer{}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	Register(r, &app.Application{
		User:     &fakeUser{permOk: true, dataScope: user.DataScope{LegalEntityID: 3, RegionScope: "root.luzon"}},
		Customer: fc, Product: &fakeProduct{}, RealName: &fakeRealName{}, CustomerLedger: &fakeLedger{},
	}, mgr)

	w := getJSON(t, r, "/api/admin/v1/customers", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if fc.lastQ.LegalEntityID != 3 || fc.lastQ.RegionScope != "root.luzon" {
		t.Fatalf("data scope not applied: lastQ=%+v", fc.lastQ)
	}
}

// TestProductListHandler 契约:产品资费列表返回 items。
func TestProductListHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	p := &fakeProduct{list: []customer.ProductOffer{
		{ID: 1, LegalEntityID: 1, Name: "500M宽带", Bandwidth: "500M", MonthlyFee: 129.00, Status: "PUBLISHED"},
	}}
	r := newCustomerRouter(&fakeCustomer{}, p, mgr)
	w := getJSON(t, r, "/api/admin/v1/products", authToken(t, mgr))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []customer.ProductOffer `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].Name != "500M宽带" || body.Data.Items[0].MonthlyFee != 129.00 {
		t.Fatalf("body=%+v", body)
	}
}

func TestCustomerCreate(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	fc := &fakeCustomer{}
	r := newCustomerRouter(fc, &fakeProduct{}, mgr)
	w := postBodyAuth(t, r, "/api/admin/v1/customers",
		`{"name":"批量导入客户","phone":"09170000001","legalEntityId":1,"addressId":1,"regionId":1}`, authToken(t, mgr))
	var env struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if w.Code != 200 || env.Code != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// TestChangeProductPrice 契约:产品调价更新月费并追加台账(worker.yaml POST /products/{id}/price-history)。
func TestChangeProductPrice(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	p := &fakeProduct{}
	r := newCustomerRouter(&fakeCustomer{}, p, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/products/1/price-history", `{"newPrice":169}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if p.changed == nil || p.changed.ID != 1 || p.changed.MonthlyFee != 169 {
		t.Fatalf("changed=%+v", p.changed)
	}
}

// TestUpdateProduct 契约:PUT /products/{id} 编辑基础信息(名称/带宽/分类),不触碰月费与状态。
func TestUpdateProduct(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	p := &fakeProduct{}
	r := newCustomerRouter(&fakeCustomer{}, p, mgr)

	w := putJSONAuth(t, r, "/api/admin/v1/products/1",
		`{"name":"1000M fusion","bandwidth":"1000M","category":"fusion"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if p.updated == nil || p.updated.ID != 1 || p.updated.Name != "1000M fusion" || p.updated.Category != "fusion" {
		t.Fatalf("updated=%+v", p.updated)
	}
	if p.changed != nil || p.statusTo != "" {
		t.Fatalf("edit must not touch fee/status: changed=%+v statusTo=%q", p.changed, p.statusTo)
	}

	// 名称缺失 → 信封 42200 参数非法(HTTP 200,信封 code 才是业务码)
	w = putJSONAuth(t, r, "/api/admin/v1/products/1", `{"bandwidth":"1G"}`, authToken(t, mgr))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":42200`) {
		t.Fatalf("missing name: status=%d body=%s", w.Code, w.Body.String())
	}
}

// TestUpdateProductStatus 契约:PUT /products/{id}/status 上下架;非法枚举 400。
func TestUpdateProductStatus(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	p := &fakeProduct{}
	r := newCustomerRouter(&fakeCustomer{}, p, mgr)

	w := putJSONAuth(t, r, "/api/admin/v1/products/1/status", `{"status":"PUBLISHED"}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if p.statusTo != "PUBLISHED" {
		t.Fatalf("statusTo=%q", p.statusTo)
	}

	w = putJSONAuth(t, r, "/api/admin/v1/products/1/status", `{"status":"PAUSED"}`, authToken(t, mgr))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":42200`) {
		t.Fatalf("invalid enum: status=%d body=%s", w.Code, w.Body.String())
	}
}
