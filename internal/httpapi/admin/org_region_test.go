package adminapi

// 回归:registerRegionRoutes 曾从 org.go 拆出后漏挂载,GET /regions 恒 404。
// 本用例走完整 Register 装配,确保路由表真实注册(而非仅 handler 可达)。

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestRegionRoutesMounted(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("未登录拒绝", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := getJSON(t, r, "/api/admin/v1/regions", "")
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want 401", w.Code)
		}
	})

	t.Run("已挂载且可访问", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := getJSON(t, r, "/api/admin/v1/regions", token)
		if w.Code == http.StatusNotFound {
			t.Fatal("GET /regions 404: registerRegionRoutes 未挂载")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", w.Code)
		}
	})

	t.Run("无菜单权限拒绝", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		req := httptest.NewRequest(http.MethodPut, "/api/admin/v1/regions/9/coverage", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status=%d, want 403", w.Code)
		}
	})
}
