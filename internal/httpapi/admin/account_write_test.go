package adminapi

// 契约:POST /accounts、PUT /accounts/{id}(menu:account)——受权建号/改号 envelope 与权限门禁;
// GET /roles(menu:account)——建号表单角色源。

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestAccountWriteHandlers(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("建号 无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := postBodyAuth(t, r, "/api/admin/v1/accounts", `{"username":"u1"}`, token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("建号 成功返回 id", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := postBodyAuth(t, r, "/api/admin/v1/accounts",
			`{"username":"ops_wang","password":"secret1","realName":"王五","roleCode":"ops"}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Code != 0 || env.Data.ID != 9 {
			t.Fatalf("env=%+v", env)
		}
	})

	t.Run("改号 成功", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := putAuth(t, r, "/api/admin/v1/accounts/9",
			`{"username":"ops_wang","realName":"王五","roleCode":"ops","status":0}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("改号 非法 id 参数错误", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := putAuth(t, r, "/api/admin/v1/accounts/abc", `{}`, token)
		var env struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code == 0 {
			t.Fatalf("expect non-zero code, body=%s", w.Body.String())
		}
	})

	t.Run("角色清单", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := getJSON(t, r, "/api/admin/v1/roles", token)
		if w.Code != 200 {
			t.Fatalf("status=%d", w.Code)
		}
		var env struct {
			Code int `json:"code"`
			Data []struct {
				Code string `json:"code"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Code != 0 || len(env.Data) != 1 || env.Data[0].Code != "ops" {
			t.Fatalf("env=%+v", env)
		}
	})
}
