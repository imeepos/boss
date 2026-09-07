package adminapi

// T2 建号接口三态契约:成功(200+CodeOK)/必填缺失与白名单外(400+42200)/唯一冲突(40900)。

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
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// fakeResourceCore 桩 resource.ResourceService:内嵌接口,仅实现被测方法。
type fakeResourceCore struct {
	resource.ResourceService
	resErr      error
	portErr     error
	createdRes  resource.Resource
	createdPort resource.Port
}

func (f *fakeResourceCore) CreateResource(_ context.Context, r resource.Resource) (int64, error) {
	f.createdRes = r
	if f.resErr != nil {
		return 0, f.resErr
	}
	return 501, nil
}

func (f *fakeResourceCore) CreatePort(_ context.Context, p resource.Port) (int64, error) {
	f.createdPort = p
	if f.portErr != nil {
		return 0, f.portErr
	}
	return 601, nil
}

func (f *fakeResourceCore) GetResource(_ context.Context, id int64) (*resource.Resource, error) {
	return &resource.Resource{ID: id, LegalEntityID: 9}, nil
}

func newOssRouter(fr *fakeResourceCore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := &app.Application{User: &fakeUser{permOk: true}, Resource: fr}
	mgr := auth.NewManager("oss-test-secret", time.Hour)
	g := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerResourceRoutes(g, a)
	return r
}

func doOss(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	tok, _ := auth.NewManager("oss-test-secret", time.Hour).Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func ossCode(t *testing.T, w *httptest.ResponseRecorder) (int, int) {
	t.Helper()
	var out struct {
		Code int `json:"code"`
	}
	_ = json.NewDecoder(w.Body).Decode(&out)
	return w.Code, out.Code
}

func TestPostResources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("建档成功:缺省 ONLINE", func(t *testing.T) {
		fr := &fakeResourceCore{}
		r := newOssRouter(fr)
		w := doOss(r, http.MethodPost, "/api/admin/v1/resources",
			`{"code":"OLT-NEW1","type":"OLT","legalEntityId":9,"addressId":3}`)
		hc, code := ossCode(t, w)
		if hc != http.StatusOK || code != int(apitypes.CodeOK) {
			t.Fatalf("http=%d code=%d body=%s", hc, code, w.Body.String())
		}
		if fr.createdRes.Status != "ONLINE" || fr.createdRes.Code != "OLT-NEW1" {
			t.Fatalf("created=%+v", fr.createdRes)
		}
	})

	t.Run("缺 addressId → 42200", func(t *testing.T) {
		r := newOssRouter(&fakeResourceCore{})
		w := doOss(r, http.MethodPost, "/api/admin/v1/resources", `{"code":"OLT-X","type":"OLT","legalEntityId":9}`)
		hc, code := ossCode(t, w)
		if hc != http.StatusBadRequest || code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("http=%d code=%d", hc, code)
		}
	})

	t.Run("type 白名单外 → 42200", func(t *testing.T) {
		r := newOssRouter(&fakeResourceCore{})
		w := doOss(r, http.MethodPost, "/api/admin/v1/resources",
			`{"code":"X-1","type":"ROUTER","legalEntityId":9,"addressId":3}`)
		_, code := ossCode(t, w)
		if code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d", code)
		}
	})

	t.Run("code 撞库 → 40900", func(t *testing.T) {
		r := newOssRouter(&fakeResourceCore{resErr: resource.ErrDuplicate})
		w := doOss(r, http.MethodPost, "/api/admin/v1/resources",
			`{"code":"OLT-01","type":"OLT","legalEntityId":9,"addressId":3}`)
		_, code := ossCode(t, w)
		if code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d body=%s", code, w.Body.String())
		}
	})
}

func TestPostPorts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("建档成功:法人快照取归属资源,缺省 IDLE", func(t *testing.T) {
		fr := &fakeResourceCore{}
		r := newOssRouter(fr)
		w := doOss(r, http.MethodPost, "/api/admin/v1/ports",
			`{"portCode":"P-NEW-01","quadCode":"P-NEW-01","resourceId":501,"addressId":3,"regionId":13,"regionName":"马尼拉"}`)
		hc, code := ossCode(t, w)
		if hc != http.StatusOK || code != int(apitypes.CodeOK) {
			t.Fatalf("http=%d code=%d body=%s", hc, code, w.Body.String())
		}
		if fr.createdPort.LegalEntityID != 9 || fr.createdPort.Status != "IDLE" {
			t.Fatalf("created=%+v", fr.createdPort)
		}
	})

	t.Run("缺 regionName → 42200", func(t *testing.T) {
		r := newOssRouter(&fakeResourceCore{})
		w := doOss(r, http.MethodPost, "/api/admin/v1/ports",
			`{"portCode":"P-2","quadCode":"P-2","resourceId":501,"addressId":3,"regionId":13}`)
		_, code := ossCode(t, w)
		if code != int(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d", code)
		}
	})

	t.Run("port_code 撞库 → 40900", func(t *testing.T) {
		r := newOssRouter(&fakeResourceCore{portErr: resource.ErrDuplicate})
		w := doOss(r, http.MethodPost, "/api/admin/v1/ports",
			`{"portCode":"P-SPL01-01","quadCode":"P-9","resourceId":501,"addressId":3,"regionId":13,"regionName":"马尼拉"}`)
		_, code := ossCode(t, w)
		if code != int(apitypes.CodeConflict) {
			t.Fatalf("code=%d", code)
		}
	})
}
