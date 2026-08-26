package userapi

// /address-tree 端点单测:children 懒加载 + 全树搜索 + 鉴权/入参拒绝。
// fakeAddressTree 以接口嵌入只实现 ListAddresses/SearchAddresses 两方法。

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeAddressTree 桩 user.Service:嵌入接口满足全量方法集,未覆盖方法被调即 panic。
type fakeAddressTree struct {
	user.Service
	children map[int64][]user.Address
	hits     []user.AddressHit
}

func (f *fakeAddressTree) ListAddresses(_ context.Context, parentID int64) ([]user.Address, error) {
	return f.children[parentID], nil
}

func (f *fakeAddressTree) SearchAddresses(_ context.Context, _ string) ([]user.AddressHit, error) {
	return f.hits, nil
}

func addressTreeFixture() *fakeAddressTree {
	ncr := user.Address{ID: 900, Level: 1, Name: "Metro Manila", CountryCode: "PH", HasChildren: true}
	qc := user.Address{ID: 901, ParentID: 900, Level: 2, Name: "Quezon City", HasChildren: true}
	bgy := user.Address{ID: 902, ParentID: 901, Level: 3, Name: "Barangay Commonwealth"}
	return &fakeAddressTree{
		children: map[int64][]user.Address{0: {ncr}, 900: {qc}, 901: {bgy}},
		hits:     []user.AddressHit{{Node: bgy, Ancestors: []user.Address{ncr, qc}}},
	}
}

// newAddressTreeRouter 装配带 user.Service 桩的路由(children/search 用例共用)。
func newAddressTreeRouter(t *testing.T, f *fakeAddressTree) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	cust := userPortalCust()
	r := gin.New()
	Register(r, &app.Application{
		Customer: &userPortalCustSvc{c: cust},
		Portal:   portal.NewMemory(),
		User:     f,
	}, mgr)
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
	return r, tok
}

func TestPortal_AddressTreeChildren_Cascade(t *testing.T) {
	r, tok := newAddressTreeRouter(t, addressTreeFixture())

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/address-tree?parentId=0", ``, tok)
	code, data := userPortalCode(t, w)
	if code != 0 {
		t.Fatalf("roots resp=%s", w.Body.String())
	}
	items, _ := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("roots items=%v", data["items"])
	}
	root, _ := items[0].(map[string]any)
	if root["name"] != "Metro Manila" || root["hasChildren"] != true {
		t.Fatalf("root=%v", root)
	}

	w = userPortalDo(r, http.MethodGet, "/api/user/v1/address-tree?parentId=900", ``, tok)
	_, data = userPortalCode(t, w)
	items, _ = data["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["name"] != "Quezon City" {
		t.Fatalf("cities items=%v", data["items"])
	}
}

func TestPortal_AddressTreeSearch(t *testing.T) {
	r, tok := newAddressTreeRouter(t, addressTreeFixture())

	w := userPortalDo(r, http.MethodGet, "/api/user/v1/address-tree/search?q=Commonwealth", ``, tok)
	code, data := userPortalCode(t, w)
	if code != 0 {
		t.Fatalf("search resp=%s", w.Body.String())
	}
	items, _ := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("hits=%v", data["items"])
	}
	hit, _ := items[0].(map[string]any)
	if hit["node"].(map[string]any)["name"] != "Barangay Commonwealth" {
		t.Fatalf("node=%v", hit)
	}
	if ans, _ := hit["ancestors"].([]any); len(ans) != 2 {
		t.Fatalf("ancestors=%v", hit["ancestors"])
	}
}

func TestPortal_AddressTreeSearch_EmptyQ(t *testing.T) {
	r, tok := newAddressTreeRouter(t, addressTreeFixture())
	w := userPortalDo(r, http.MethodGet, "/api/user/v1/address-tree/search", ``, tok)
	if code, _ := userPortalCode(t, w); code == 0 {
		t.Fatalf("empty q should fail: %s", w.Body.String())
	}
}

func TestPortal_AddressTree_Unauthorized(t *testing.T) {
	r, _ := newAddressTreeRouter(t, addressTreeFixture())
	w := userPortalDo(r, http.MethodGet, "/api/user/v1/address-tree?parentId=0", ``, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no token status=%d", w.Code)
	}
}
