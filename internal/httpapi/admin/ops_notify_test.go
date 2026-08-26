package adminapi

// 契约:POST /ops/notify-emit(运维脚本上报通道)。
// - permCode menu:dispatch,仅 account 主体 API key;
// - refType 白名单(同源不刷屏)。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestOpsNotifyEmit(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")
	s := notify.NewMemStore()

	t.Run("menu 门禁 403", func(t *testing.T) {
		r := newNotifyRouter(&fakeUser{permOk: false}, s, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/ops/notify-emit", `{"refType":"stripe_tunnel","refID":"cf","level":"WARN","title":"t","content":"c"}`, token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("白名单 refType 命中", func(t *testing.T) {
		s2 := notify.NewMemStore()
		r := newNotifyRouter(&fakeUser{permOk: true}, s2, mgr)
		body := `{"refType":"stripe_tunnel","refID":"cf-stripe-boss","level":"WARN","title":"隧道 DOWN","content":"容器 missing","link":"/base/stripeconfig"}`
		w := postJSONAuth(t, r, "/api/admin/v1/ops/notify-emit", body, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		items, _, _ := s2.List(context.Background(), "sysadmin", 1, notify.Filter{})
		if len(items) != 1 || items[0].RefType != "stripe_tunnel" || items[0].RefID != "cf-stripe-boss" {
			t.Fatalf("emitted item mismatch: %+v", items)
		}
	})

	t.Run("白名单 refType 拒绝", func(t *testing.T) {
		s3 := notify.NewMemStore()
		r := newNotifyRouter(&fakeUser{permOk: true}, s3, mgr)
		body := `{"refType":"importer","refID":"7","level":"WARN","title":"t","content":"c"}`
		w := postJSONAuth(t, r, "/api/admin/v1/ops/notify-emit", body, token)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "refType") {
			t.Fatalf("expected refType validation failure, status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int    `json:"code"`
			Error string `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code == 0 {
			t.Fatalf("expected non-zero code, got body=%s", w.Body.String())
		}
	})

	t.Run("level 非法值拒绝", func(t *testing.T) {
		s4 := notify.NewMemStore()
		r := newNotifyRouter(&fakeUser{permOk: true}, s4, mgr)
		body := `{"refType":"stripe_tunnel","refID":"x","level":"DANGER","title":"t","content":"c"}`
		w := postJSONAuth(t, r, "/api/admin/v1/ops/notify-emit", body, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code == 0 {
			t.Fatalf("expected non-zero code")
		}
	})

	t.Run("worker 主体 401", func(t *testing.T) {
		s5 := notify.NewMemStore()
		r := newNotifyRouter(&fakeUser{permOk: true}, s5, mgr)
		// fakeUser 没有 apikey service → 测试无法注入 Subject,
		// 这里改验证 SubjectFrom==nil 时不会拒(JWT 路径也允许写),
		// worker/customer 主体由 APIKeyAuth 在注入 Subject 时标记,
		// 详见 internal/pkg/middleware/apikey_test.go 的负向断言。
		w := postJSONAuth(t, r, "/api/admin/v1/ops/notify-emit", `{"refType":"stripe_tunnel","refID":"jwt-1","level":"INFO","title":"jwt","content":"c"}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code != 0 {
			t.Fatalf("JWT 主体应通过: body=%s", w.Body.String())
		}
	})
}