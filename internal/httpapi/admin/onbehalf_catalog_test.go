package adminapi

// 代客受理目录测试:/orders/catalog 与 /customers/onboarding-catalog。
// 口径:产品只回在售;地址搜索仅在带 q 时返回,拍平祖先链为 fullPath。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeChannelSvc 渠道目录桩。
type fakeChannelSvc struct {
	list []order.Channel
}

func (f *fakeChannelSvc) ListChannels(context.Context) ([]order.Channel, error) {
	return f.list, nil
}
func (f *fakeChannelSvc) GetChannel(_ context.Context, id int64) (*order.Channel, error) {
	for i := range f.list {
		if f.list[i].ID == id {
			return &f.list[i], nil
		}
	}
	return nil, nil
}
func (f *fakeChannelSvc) CreateChannel(context.Context, order.Channel) (int64, error) {
	return 0, nil
}

func newCatalogRouter(u *fakeUser, p *fakeProduct, ch *fakeChannelSvc) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	r := gin.New()
	ja := &app.Application{User: u, Product: p, Channel: ch}
	Register(r, ja, mgr)
	return r, mgr
}

// TestOrderCatalog_ProductsFilteredProductsAndChannels 在售过滤 + 渠道目录。
func TestOrderCatalog_ProductsFilteredProductsAndChannels(t *testing.T) {
	u := &fakeUser{permOk: true}
	p := &fakeProduct{list: []customer.ProductOffer{
		{ID: 1, Name: "在售套餐", Status: "PUBLISHED"},
		{ID: 2, Name: "草稿套餐", Status: "DRAFT"},
		{ID: 3, Name: "下架套餐", Status: "OFFLINE"},
	}}
	ch := &fakeChannelSvc{list: []order.Channel{{ID: 5, Code: "HALL", Name: "营业厅", Status: "ACTIVE"}}}
	r, mgr := newCatalogRouter(u, p, ch)

	w := getJSON(t, r, "/api/admin/v1/orders/catalog", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Products []customer.ProductOffer `json:"products"`
			Channels []order.Channel         `json:"channels"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data.Products) != 1 || resp.Data.Products[0].ID != 1 {
		t.Fatalf("只应回在售产品: %+v", resp.Data.Products)
	}
	if len(resp.Data.Channels) != 1 || resp.Data.Channels[0].ID != 5 {
		t.Fatalf("渠道目录缺失: %+v", resp.Data.Channels)
	}
	var raw struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if _, has := raw.Data["addresses"]; has {
		t.Fatalf("无 q 不应回地址: %s", w.Body.String())
	}
}

// TestOrderCatalog_AddressSearch 带 q 附地址搜索,祖先链拍平 fullPath。
func TestOrderCatalog_AddressSearch(t *testing.T) {
	u := &fakeUser{permOk: true, addrHits: []user.AddressHit{{
		Node:      user.Address{ID: 104, Name: "小区A"},
		Ancestors: []user.Address{{ID: 1, Name: "市"}, {ID: 2, Name: "区"}},
	}}}
	r, mgr := newCatalogRouter(u, &fakeProduct{}, &fakeChannelSvc{})

	w := getJSON(t, r, "/api/admin/v1/orders/catalog?q=%E5%B0%8F%E5%8C%BA", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Addresses []catalogAddress `json:"addresses"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data.Addresses) != 1 {
		t.Fatalf("地址搜索缺失: %+v", resp.Data)
	}
	got := resp.Data.Addresses[0]
	if got.ID != 104 || got.FullPath != "市 / 区 / 小区A" {
		t.Fatalf("fullPath 拍平错误: %+v", got)
	}
}

// TestCustomerOnboardingCatalog 区域+主体目录。
func TestCustomerOnboardingCatalog(t *testing.T) {
	u := &fakeUser{permOk: true,
		regions:  []user.Region{{ID: 4, Name: "马尼拉", Level: 4}},
		entities: []user.LegalEntity{{ID: 1, Code: "LEG-A", Name: "总公司", IsPlatform: true}},
	}
	r, mgr := newCatalogRouter(u, &fakeProduct{}, &fakeChannelSvc{})

	w := getJSON(t, r, "/api/admin/v1/customers/onboarding-catalog", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Regions       []user.Region      `json:"regions"`
			LegalEntities []user.LegalEntity `json:"legalEntities"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data.Regions) != 1 || resp.Data.Regions[0].ID != 4 {
		t.Fatalf("区域目录缺失: %+v", resp.Data.Regions)
	}
	if len(resp.Data.LegalEntities) != 1 || resp.Data.LegalEntities[0].ID != 1 {
		t.Fatalf("主体目录缺失: %+v", resp.Data.LegalEntities)
	}
}

// TestOnbehalfCatalog_NoPermission 无受理域权限拒绝(403)。
func TestOnbehalfCatalog_NoPermission(t *testing.T) {
	u := &fakeUser{permOk: false}
	r, mgr := newCatalogRouter(u, &fakeProduct{}, &fakeChannelSvc{})
	tok := authToken(t, mgr)

	for _, path := range []string{"/api/admin/v1/orders/catalog", "/api/admin/v1/customers/onboarding-catalog"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s code=%d body=%s", path, w.Code, w.Body.String())
		}
	}
}
