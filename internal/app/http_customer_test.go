package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakeCustomer struct{ list []customer.Customer }

func (f *fakeCustomer) Create(context.Context, customer.Customer) (int64, error) { return 0, nil }
func (f *fakeCustomer) Get(context.Context, int64) (*customer.Customer, error)   { return nil, nil }
func (f *fakeCustomer) List(context.Context, customer.CustomerQuery) ([]customer.Customer, error) {
	return f.list, nil
}

type fakeProduct struct {
	list []customer.ProductOffer
	changed *customer.ProductOffer // 记录最近一次调价入参
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

type fakeRealName struct{}

func (f *fakeRealName) ListVerifications(context.Context, int64) ([]customer.RealNameVerification, error) {
	return nil, nil
}
func (f *fakeRealName) AppendVerification(context.Context, customer.RealNameVerification) (int64, error) {
	return 0, nil
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
	RegisterRoutes(r, &Application{
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
	w := getJSON(t, r, "/api/v1/customers", authToken(t, mgr))

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
}

// TestProductListHandler 契约:产品资费列表返回 items。
func TestProductListHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	p := &fakeProduct{list: []customer.ProductOffer{
		{ID: 1, LegalEntityID: 1, Name: "500M宽带", Bandwidth: "500M", MonthlyFee: 129.00, Status: "PUBLISHED"},
	}}
	r := newCustomerRouter(&fakeCustomer{}, p, mgr)
	w := getJSON(t, r, "/api/v1/products", authToken(t, mgr))

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

// TestChangeProductPrice 契约:产品调价更新月费并追加台账(worker.yaml POST /products/{id}/price-history)。
func TestChangeProductPrice(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	p := &fakeProduct{}
	r := newCustomerRouter(&fakeCustomer{}, p, mgr)

	w := postBodyAuth(t, r, "/api/v1/products/1/price-history", `{"newPrice":169}`, authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if p.changed == nil || p.changed.ID != 1 || p.changed.MonthlyFee != 169 {
		t.Fatalf("changed=%+v", p.changed)
	}
}
