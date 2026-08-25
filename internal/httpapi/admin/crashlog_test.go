package adminapi

// 契约:GET /crash-logs(menu:crash_logs);列表按 created_at DESC,limit 1..200 默认 50。

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	crashdomain "github.com/ymm-001/boss/internal/domain/crash"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// newCrashRouter 仅构造 crash-logs 路由,避免全 Register 拉起其它依赖。
// 鉴权走 admin 标准链:API key 免登录 + JWT(admin aud);权限码 menu:crash_logs。
func newCrashRouter(store crashdomain.Store, permOk bool, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	f := &fakeUser{permOk: permOk}
	a := &app.Application{User: f, CrashLogs: store}
	authed := r.Group("/api/admin/v1", middleware.Authn(mgr, auth.AudAdmin))
	registerCrashLogRoutes(authed, a)
	return r
}

func TestCrashLogsList(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	tok, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("perm 通过 + 列表返回", func(t *testing.T) {
		store := crashdomain.NewMemStore()
		_ = store.Insert(context.Background(), crashdomain.Log{App: "boss-worker/0.1.0", Log: "thread: main\nstack: x"})
		_ = store.Insert(context.Background(), crashdomain.Log{App: "boss-worker/0.1.0", Log: "thread: ui\nstack: y"})

		r := newCrashRouter(store, true, mgr)
		w := getJSON(t, r, "/api/admin/v1/crash-logs", tok)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int               `json:"code"`
			Data []crashdomain.Log `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code != 0 {
			t.Fatalf("code=%d body=%s", env.Code, w.Body.String())
		}
		if len(env.Data) != 2 {
			t.Fatalf("want 2, got %d", len(env.Data))
		}
	})

	t.Run("limit 越界回退默认", func(t *testing.T) {
		store := crashdomain.NewMemStore()
		_ = store.Insert(context.Background(), crashdomain.Log{Log: "x"})
		r := newCrashRouter(store, true, mgr)
		w := getJSON(t, r, "/api/admin/v1/crash-logs?limit=9999", tok)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d", w.Code)
		}
	})

	t.Run("无 permCode 返回 403", func(t *testing.T) {
		store := crashdomain.NewMemStore()
		r := newCrashRouter(store, false, mgr)
		w := getJSON(t, r, "/api/admin/v1/crash-logs", tok)
		if w.Code != http.StatusForbidden {
			t.Fatalf("want 403, got %d body=%s", w.Code, w.Body.String())
		}
	})
}
