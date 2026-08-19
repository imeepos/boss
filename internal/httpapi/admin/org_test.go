package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func getJSON(t *testing.T, r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestOrgListHandler 契约:组织列表需登录(RBAC 菜单权限码),越权/未登录被拒。
func TestOrgListHandler(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(1, "boss", "sysadmin")

	t.Run("未登录", func(t *testing.T) {
		f := &fakeUser{}
		r := newTestRouter(f, mgr)
		w := getJSON(t, r, "/api/v1/legal-entities", "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", w.Code)
		}
	})

	t.Run("无权限", func(t *testing.T) {
		f := &fakeUser{permOk: false}
		r := newTestRouter(f, mgr)
		w := getJSON(t, r, "/api/v1/legal-entities", token)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status=%d, want 403", w.Code)
		}
	})

	t.Run("有权限", func(t *testing.T) {
		f := &fakeUser{
			permOk:   true,
			entities: []user.LegalEntity{{ID: 1, Code: "LEG-A", Name: "主品牌"}},
		}
		r := newTestRouter(f, mgr)
		w := getJSON(t, r, "/api/v1/legal-entities", token)

		if w.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", w.Code)
		}
		var body struct {
			Code int `json:"code"`
			Data []struct {
				ID   int64  `json:"id"`
				Code string `json:"code"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Code != 0 || len(body.Data) != 1 || body.Data[0].Code != "LEG-A" {
			t.Fatalf("body=%+v", body)
		}
	})
}
