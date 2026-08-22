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
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// fakeProvSeed 置备桩:资源/渠道/资产/派单各 create 口;计数供断言。
type fakeProvSeed struct {
	resource.ResourceService
	order.ChannelService
	asset.AssetService
	order.WorkOrderService

	resID, portID, chanID, batchID, tagID, assetID, tktID int64
}

func (f *fakeProvSeed) CreateResource(context.Context, resource.Resource) (int64, error) {
	f.resID++
	return f.resID, nil
}
func (f *fakeProvSeed) CreatePort(context.Context, resource.Port) (int64, error) {
	f.portID++
	return f.portID, nil
}
func (f *fakeProvSeed) CreateChannel(context.Context, order.Channel) (int64, error) {
	f.chanID++
	return f.chanID, nil
}

// ListChannels GET 目录桩:固定两条,供列表断言。
func (f *fakeProvSeed) ListChannels(context.Context) ([]order.Channel, error) {
	return []order.Channel{
		{ID: 1, Code: "HALL", Name: "营业厅", Status: "ACTIVE"},
		{ID: 2, Code: "ONLINE", Name: "线上", Status: "ACTIVE"},
	}, nil
}
func (f *fakeProvSeed) CreateBatch(context.Context, asset.AssetBatch) (int64, error) {
	f.batchID++
	return f.batchID, nil
}
func (f *fakeProvSeed) CreateTag(context.Context, asset.Tag) (int64, error) {
	f.tagID++
	return f.tagID, nil
}
func (f *fakeProvSeed) CreateAsset(context.Context, asset.Asset) (int64, error) {
	f.assetID++
	return f.assetID, nil
}
func (f *fakeProvSeed) CreateDispatchTicket(context.Context, order.DispatchTicket) (int64, error) {
	f.tktID++
	return f.tktID, nil
}

func newProvSeedRouter(f *fakeProvSeed, mgr *auth.Manager) (*gin.Engine, *auth.Manager) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{
		User:       &fakeUser{permOk: true},
		Resource:   f,
		Channel:    f,
		Asset:      f,
		WorkOrder:  f,
		Order:      &fakeOrder{},
		Automation: app.NewAutomation(&fakeOrder{}, nil),
	}, mgr)
	return r, mgr
}

func TestProvisionSeedAll(t *testing.T) {
	f := &fakeProvSeed{}
	r, mgr := newProvSeedRouter(f, auth.NewManager("s", time.Hour))
	tok := authToken(t, mgr)

	cases := []struct{ path, body string }{
		{"/api/admin/v1/provision/resources", `{"code":"OLT-01","name":"OLT","type":"OLT","addressId":288,"legalEntityId":1}`},
		{"/api/admin/v1/provision/ports", `{"portCode":"P-01","resourceId":1,"addressId":288,"legalEntityId":1}`},
		{"/api/admin/v1/provision/channels", `{"code":"HALL","name":"营业厅","status":"ACTIVE"}`},
		{"/api/admin/v1/provision/asset-batches", `{"code":"RK-01","name":"批次","legalEntityId":1}`},
		{"/api/admin/v1/provision/tags", `{"tagNo":"T1","epcCode":"EPC-1","legalEntityId":1}`},
		{"/api/admin/v1/provision/assets", `{"assetCode":"A-01","batchId":1,"tagId":1,"legalEntityId":1}`},
		{"/api/admin/v1/provision/dispatch-tickets", `{"ticketNo":"TK-1","orderId":333}`},
	}
	for _, tc := range cases {
		w := postBodyAuth(t, r, tc.path, tc.body, tok)
		if w.Code != 200 {
			t.Fatalf("%s code=%d body=%s", tc.path, w.Code, w.Body.String())
		}
	}
	if f.resID != 1 || f.portID != 1 || f.chanID != 1 || f.batchID != 1 ||
		f.tagID != 1 || f.assetID != 1 || f.tktID != 1 {
		t.Fatalf("置备计数不一致: %+v", f)
	}
}

// TestProvisionSeed_Invalid 缺必填字段 → 非 200 业务码,且不触发 create。
func TestProvisionSeed_Invalid(t *testing.T) {
	f := &fakeProvSeed{}
	r, mgr := newProvSeedRouter(f, auth.NewManager("s", time.Hour))
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/provision/dispatch-tickets", `{"ticketNo":"TK-1"}`, tok)
	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 0 || f.tktID != 0 {
		t.Fatalf("应拒绝缺 orderId: resp=%+v", resp)
	}
}

// TestProvisionListChannels GET /provision/channels 渠道目录(ISSUE.md 渠道无查询接口)。
func TestProvisionListChannels(t *testing.T) {
	f := &fakeProvSeed{}
	r, mgr := newProvSeedRouter(f, auth.NewManager("s", time.Hour))
	tok := authToken(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/provision/channels", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var out struct {
		Code int `json:"code"`
		Data struct {
			Items []order.Channel `json:"items"`
			Total int             `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Code != 0 || out.Data.Total != 2 || len(out.Data.Items) != 2 || out.Data.Items[0].Code != "HALL" {
		t.Fatalf("list channels resp=%s", w.Body.String())
	}
}
