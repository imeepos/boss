package adminapi

// 契约:GET /docs/openapi(menu:apidocs)——聚合契约 JSON 下发;
// portal 非法 401 参数错;无权限 403(api/openapi/admin/docs.yaml)。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func TestApiDocsSpec(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, err := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	r := newTestRouter(&fakeUser{permOk: true}, mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/docs/openapi?portal=admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			OpenAPI string         `json:"openapi"`
			Paths   map[string]any `json:"paths"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != 0 || body.Data.OpenAPI != "3.0.3" || len(body.Data.Paths) == 0 {
		t.Fatalf("code=%d openapi=%q paths=%d", body.Code, body.Data.OpenAPI, len(body.Data.Paths))
	}
}

func TestApiDocsSpecGate(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("portal 非法", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/docs/openapi?portal=bad", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var body struct {
			Code int32 `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if body.Code != int32(apitypes.CodeInvalidParam) {
			t.Fatalf("code=%d, want %d", body.Code, apitypes.CodeInvalidParam)
		}
	})

	t.Run("无权限", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/docs/openapi", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var body struct {
			Code int32 `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		// 门禁中间件写裸 403(非 apitypes.CodeForbidden),与全站权限拒绝行为一致。
		if body.Code != 403 {
			t.Fatalf("code=%d, want 403", body.Code)
		}
	})
}
