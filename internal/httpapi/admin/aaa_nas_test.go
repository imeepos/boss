package adminapi

// AAA-A5 契约:NAS 注册表 CRUD(menu:loaccount)——密钥仅写不回显;IP 重复 40900 语义由域错误透传。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type nasStubAaa struct {
	fakeAaa
	id      int64
	upsert  aaa.NasUpsert
	created bool
	updated bool
	deleted bool
}

func (s *nasStubAaa) CreateNas(_ context.Context, u aaa.NasUpsert) (int64, error) {
	s.created, s.upsert = true, u
	return 77, nil
}

func (s *nasStubAaa) UpdateNas(_ context.Context, _ int64, u aaa.NasUpsert) error {
	s.updated, s.upsert = true, u
	return nil
}

func (s *nasStubAaa) DeleteNas(context.Context, int64) error { s.deleted = true; return nil }

func (s *nasStubAaa) GetNas(_ context.Context, id int64) (*aaa.NasClient, error) {
	return &aaa.NasClient{ID: id, Name: "OLT-01", NasIP: "10.0.0.9", Vendor: aaa.NasVendorHuawei, CoAPort: 3799, Enabled: true}, nil
}

func (s *nasStubAaa) ListNasPage(context.Context, aaa.NasPage) (aaa.AdminPageResult[aaa.NasClient], error) {
	return aaa.AdminPageResult[aaa.NasClient]{Total: 1}, nil
}

// doNasAuth 带鉴权令牌发起任意方法 JSON 请求。
func doNasAuth(t *testing.T, r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newNasRouter(aa aaa.AaaService, permOk bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("nas-test-secret", time.Hour)
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: permOk}, Aaa: aa}, mgr)
	return r
}

func nasToken(t *testing.T) string {
	t.Helper()
	tok, err := auth.NewManager("nas-test-secret", time.Hour).Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestNasCreateRejectsEmptySecret(t *testing.T) {
	r := newNasRouter(&nasStubAaa{}, true)
	w := doNasAuth(t, r, http.MethodPost, "/api/admin/v1/aaa/nas", nasToken(t), `{"name":"OLT","nasIp":"10.0.0.9"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestNasCreateNeverEchoesSecret(t *testing.T) {
	stub := &nasStubAaa{}
	r := newNasRouter(stub, true)
	w := doNasAuth(t, r, http.MethodPost, "/api/admin/v1/aaa/nas", nasToken(t),
		`{"name":"OLT-01","nasIp":"10.0.0.9","secret":"PlainSecret9","vendor":"HUAWEI","coaPort":3799,"enabled":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !stub.created || stub.upsert.SecretPlain != "PlainSecret9" {
		t.Fatalf("stub=%+v", stub.upsert)
	}
	if strings.Contains(w.Body.String(), "PlainSecret9") {
		t.Fatal("响应不得回显密钥明文")
	}
}

func TestNasUpdateDeleteFlow(t *testing.T) {
	stub := &nasStubAaa{}
	r := newNasRouter(stub, true)
	tok := nasToken(t)
	w := doNasAuth(t, r, http.MethodPut, "/api/admin/v1/aaa/nas/77", tok, `{"name":"OLT-01","nasIp":"10.0.0.9","enabled":false}`)
	if w.Code != http.StatusOK || !stub.updated {
		t.Fatalf("update: status=%d body=%s", w.Code, w.Body.String())
	}
	w = doNasAuth(t, r, http.MethodDelete, "/api/admin/v1/aaa/nas/77", tok, "")
	if w.Code != http.StatusOK || !stub.deleted {
		t.Fatalf("delete: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestNasRoutesRequirePermission(t *testing.T) {
	r := newNasRouter(&nasStubAaa{}, false)
	tok := nasToken(t)
	for _, tc := range []struct {
		method, path string
	}{{http.MethodGet, "/api/admin/v1/aaa/nas"}, {http.MethodPost, "/api/admin/v1/aaa/nas"},
		{http.MethodGet, "/api/admin/v1/aaa/nas/77"}, {http.MethodPut, "/api/admin/v1/aaa/nas/77"},
		{http.MethodDelete, "/api/admin/v1/aaa/nas/77"}} {
		w := doNasAuth(t, r, tc.method, tc.path, tok, `{}`)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s %s: status=%d want 403", tc.method, tc.path, w.Code)
		}
	}
}
