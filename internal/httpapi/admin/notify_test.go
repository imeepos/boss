package adminapi

// 契约:GET /notifications、GET /notifications/unread-count、POST /notifications/read
// (menu:dispatch 门禁,docs/plan/admin-notify-center.md §3)。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func newNotifyRouter(f *fakeUser, s notify.Service, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: f, Notify: s}, mgr)
	return r
}

func TestNotifyRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	seed := func() notify.Service {
		s := notify.NewMemStore()
		_ = s.Emit(context.Background(), notify.Input{
			Category: notify.CategoryTask, Level: notify.LevelWarn,
			Title: "导入失败", RefType: "importer", RefID: "7",
		})
		_ = s.Emit(context.Background(), notify.Input{
			Category: notify.CategoryTodo, Title: "师傅注册待审核",
			RefType: "worker_reg", RefID: "9",
		})
		return s
	}

	t.Run("无权限 403", func(t *testing.T) {
		r := newNotifyRouter(&fakeUser{permOk: false}, seed(), mgr)
		w := getJSON(t, r, "/api/admin/v1/notifications", token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("清单+过滤", func(t *testing.T) {
		r := newNotifyRouter(&fakeUser{permOk: true}, seed(), mgr)
		w := getJSON(t, r, "/api/admin/v1/notifications?category=todo", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
			Data struct {
				Items []notify.Item `json:"items"`
				Total int           `json:"total"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code != 0 || env.Data.Total != 1 || len(env.Data.Items) != 1 {
			t.Fatalf("env=%+v", env)
		}
		if env.Data.Items[0].RefType != "worker_reg" || env.Data.Items[0].Read {
			t.Fatalf("item=%+v", env.Data.Items[0])
		}
	})

	t.Run("未读数+全部已读", func(t *testing.T) {
		r := newNotifyRouter(&fakeUser{permOk: true}, seed(), mgr)
		w := getJSON(t, r, "/api/admin/v1/notifications/unread-count", token)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"count":2`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		w2 := postJSONAuth(t, r, "/api/admin/v1/notifications/read", `{"ids":[]}`, token)
		if w2.Code != 200 {
			t.Fatalf("status=%d body=%s", w2.Code, w2.Body.String())
		}
		w3 := getJSON(t, r, "/api/admin/v1/notifications/unread-count", token)
		if !strings.Contains(w3.Body.String(), `"count":0`) {
			t.Fatalf("after mark read body=%s", w3.Body.String())
		}
	})
}
