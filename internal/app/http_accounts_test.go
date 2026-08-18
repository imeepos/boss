package app

// 契约:GET /accounts(menu:account 权限码)——登录+授权后返回账号列表 envelope。

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestAccountsListHandler(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(1, "boss", "sysadmin")

	t.Run("无权限 403", func(t *testing.T) {
		f := &fakeUser{permOk: false}
		r := newTestRouter(f, mgr)
		w := getJSON(t, r, "/api/v1/accounts", token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("有权限返回列表", func(t *testing.T) {
		f := &fakeUser{permOk: true, accounts: []user.AccountRow{
			{ID: 1, Username: "admin", RealName: "管理员", RoleCode: "sysadmin", RoleName: "系统管理员", Status: 1},
		}}
		r := newTestRouter(f, mgr)
		w := getJSON(t, r, "/api/v1/accounts", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int               `json:"code"`
			Data []user.AccountRow `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Code != 0 || len(env.Data) != 1 || env.Data[0].Username != "admin" {
			t.Fatalf("env=%+v", env)
		}
	})
}
