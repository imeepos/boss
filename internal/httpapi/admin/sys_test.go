package adminapi

// 契约:GET /audit-logs(menu:audit)与 GET/PUT /params(menu:params)——sys 横切受权端点。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func newTestRouterWithAudit(f *fakeUser, aw audit.Writer, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: f, Audit: aw}, mgr)
	return r
}

func jsonContains(s, sub string) bool { return strings.Contains(s, sub) }

type fakeAudit struct{ entries []audit.Entry }

func (f *fakeAudit) Write(ctx context.Context, e audit.Event) error { return nil }
func (f *fakeAudit) List(ctx context.Context, q audit.Query) ([]audit.Entry, error) {
	return f.entries, nil
}

func TestSysRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("审计日志 无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := getJSON(t, r, "/api/admin/v1/audit-logs", token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("审计日志 返回 items+操作人", func(t *testing.T) {
		entries := []audit.Entry{{
			ID: 1, AccountID: 1, Operator: "管理员", Action: "权限变更",
			TargetType: "account", TargetID: "9", Detail: "{}", IP: "10.0.0.1",
			CreatedAt: time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC),
		}}
		r := newTestRouterWithAudit(&fakeUser{permOk: true}, &fakeAudit{entries: entries}, mgr)
		w := getJSON(t, r, "/api/admin/v1/audit-logs", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
			Data struct {
				Items []audit.Entry `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code != 0 || len(env.Data.Items) != 1 || env.Data.Items[0].Operator != "管理员" {
			t.Fatalf("env=%+v", env)
		}
	})

	t.Run("业务参数 清单与热更", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := getJSON(t, r, "/api/admin/v1/params", token)
		if w.Code != 200 {
			t.Fatalf("status=%d", w.Code)
		}
		w2 := putAuth(t, r, "/api/admin/v1/params/arrears.threshold", `{"value":"100"}`, token)
		if w2.Code != 200 {
			t.Fatalf("status=%d body=%s", w2.Code, w2.Body.String())
		}
	})

	t.Run("导入任务清单 有权限 200", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := getJSON(t, r, "/api/admin/v1/import-tasks", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("导入结果登记 合法 kind 落库", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := postBodyAuth(t, r, "/api/admin/v1/import-tasks",
			`{"kind":"entity:department","total":4,"imported":2,"failed":1,"skipped":1}`, token)
		if w.Code != 200 || !jsonContains(w.Body.String(), `"ok":true`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("导入结果登记 非法 kind/负数 拒绝", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		for _, body := range []string{
			`{"kind":"addresses"}`, `{"kind":"entity:Drop Table"}`, `{"kind":"entity:LegalEntity"}`, `{"kind":"entity:ok","total":1,"imported":-1}`,
		} {
			w := postBodyAuth(t, r, "/api/admin/v1/import-tasks", body, token)
			var env struct {
				Code int `json:"code"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &env)
			if w.Code != 200 || env.Code == 0 {
				t.Fatalf("body=%s status=%d env=%+v", body, w.Code, env)
			}
		}
	})

	t.Run("导入结果登记 无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := postBodyAuth(t, r, "/api/admin/v1/import-tasks", `{"kind":"entity:post"}`, token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("业务参数 无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := putAuth(t, r, "/api/admin/v1/params/arrears.threshold", `{"value":"100"}`, token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})
}
