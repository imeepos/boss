package adminapi

// 聚合搜索 handler 测试:关键字必填 / 四域分组 / 域权限过滤 / 数据范围透传。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// permUser 内嵌 fakeUser:按权限码集合判定,缺省全无权限。
type permUser struct {
	fakeUser
	perms map[string]bool
}

func (p *permUser) HasPermission(_ context.Context, _ int64, code string) (bool, error) {
	return p.perms[code], nil
}

// searchWorker 内嵌 fakeWorkerOps:ListWorkers 返回可配置列表。
type searchWorker struct {
	fakeWorkerOps
	list []worker.Worker
}

func (s *searchWorker) ListWorkers(_ context.Context, _ int64, _ string) ([]worker.Worker, error) {
	return s.list, nil
}

func newSearchRouter(fu *permUser, fc *fakeCustomer, fo *fakeOrder, fw *searchWorker, fud *fakeUserdata) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User: fu, Customer: fc, Order: fo, Worker: fw, UserData: fud,
	}, auth.NewManager("s", time.Hour))
	return r
}

// searchGroupResp 聚合响应分组解析骨架(items 逐域字段不同,此处仅断言条数)。
type searchGroupResp struct {
	Domain string            `json:"domain"`
	Items  []json.RawMessage `json:"items"`
}

func decodeGroups(t *testing.T, body []byte) []searchGroupResp {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Groups []searchGroupResp `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("code=%d", resp.Code)
	}
	return resp.Data.Groups
}

func allPerms() map[string]bool {
	return map[string]bool{
		"menu:customer": true, "menu:user": true, "menu:dispatch": true, "menu:order": true,
	}
}

// TestSearchRequiresKeyword 契约:缺 keyword 回 InvalidParam。
func TestSearchRequiresKeyword(t *testing.T) {
	r := newSearchRouter(&permUser{perms: allPerms()}, &fakeCustomer{}, &fakeOrder{}, &searchWorker{}, &fakeUserdata{})
	w := getJSON(t, r, "/api/admin/v1/search", authToken(t, auth.NewManager("s", time.Hour)))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != int(apitypes.CodeInvalidParam) {
		t.Fatalf("code=%d", resp.Code)
	}
}

// TestSearchGroupsAllDomains 契约:四域全有权限时按声明顺序返回四组,各含命中。
func TestSearchGroupsAllDomains(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	fc := &fakeCustomer{list: []customer.Customer{
		{ID: 1, Name: "王先生", Phone: "13800001111", CustomerCode: "C-1", ServiceStatus: "ACTIVE"},
	}}
	fud := &fakeUserdata{row: map[string]any{"customerId": int64(2), "name": "王先生", "phone": "13800001111"}}
	fw := &searchWorker{list: []worker.Worker{{ID: 3, StaffNo: "W001", Name: "李师傅", Phone: "13900002222", Status: 1}}}
	fo := &fakeOrder{list: []order.OrderListItem{{ID: 4, OrderNo: "ORD-1", Customer: "王先生", Product: "100M", Status: "PENDING"}}}
	r := newSearchRouter(&permUser{perms: allPerms()}, fc, fo, fw, fud)

	w := getJSON(t, r, "/api/admin/v1/search?keyword=%E7%8E%8B", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	groups := decodeGroups(t, w.Body.Bytes())
	if len(groups) != 4 {
		t.Fatalf("groups=%d, want 4", len(groups))
	}
	want := []string{"customer", "user", "worker", "order"}
	for i, g := range groups {
		if g.Domain != want[i] {
			t.Fatalf("groups[%d].domain=%s, want %s", i, g.Domain, want[i])
		}
		if len(g.Items) != 1 {
			t.Fatalf("groups[%d].items=%d, want 1", i, len(g.Items))
		}
	}
}

// TestSearchFiltersDomainsByPerm 契约:仅持部分域权限时,缺权限域不检索不出组。
func TestSearchFiltersDomainsByPerm(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	fc := &fakeCustomer{list: []customer.Customer{{ID: 1, Name: "王先生"}}}
	fo := &fakeOrder{list: []order.OrderListItem{{ID: 4, OrderNo: "ORD-1"}}}
	// user/worker 域同样有数据,但账号无对应权限。
	fud := &fakeUserdata{row: map[string]any{"customerId": int64(2), "name": "王先生"}}
	fw := &searchWorker{list: []worker.Worker{{ID: 3, StaffNo: "W001", Name: "李师傅"}}}
	perms := map[string]bool{"menu:customer": true, "menu:order": true}
	r := newSearchRouter(&permUser{perms: perms}, fc, fo, fw, fud)

	w := getJSON(t, r, "/api/admin/v1/search?keyword=wang", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	groups := decodeGroups(t, w.Body.Bytes())
	if len(groups) != 2 {
		t.Fatalf("groups=%d, want 2", len(groups))
	}
	if groups[0].Domain != "customer" || groups[1].Domain != "order" {
		t.Fatalf("domains=%s,%s", groups[0].Domain, groups[1].Domain)
	}
}

// TestSearchAppliesDataScope 契约:客户/订单域透传账号数据范围。
func TestSearchAppliesDataScope(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	fc := &fakeCustomer{list: []customer.Customer{{ID: 1, Name: "王先生"}}}
	fo := &fakeOrder{list: []order.OrderListItem{{ID: 4, OrderNo: "ORD-1"}}}
	fu := &permUser{perms: allPerms(), fakeUser: fakeUser{dataScope: user.DataScope{LegalEntityID: 3, RegionScope: "root.luzon"}}}
	r := newSearchRouter(fu, fc, fo, &searchWorker{}, &fakeUserdata{})

	w := getJSON(t, r, "/api/admin/v1/search?keyword=wang", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if fc.lastQ.LegalEntityID != 3 || fc.lastQ.RegionScope != "root.luzon" {
		t.Fatalf("customer scope=%+v", fc.lastQ)
	}
	if fo.lastQ.LegalEntityID != 3 || fo.lastQ.RegionScope != "root.luzon" {
		t.Fatalf("order scope=%+v", fo.lastQ)
	}
}
