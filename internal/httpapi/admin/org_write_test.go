package adminapi

// 契约:POST/PUT /departments、/posts(menu:department/menu:post)——基础数据受权维护 envelope 与门禁。

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestOrgWriteHandlers(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(1, "boss", "sysadmin")

	t.Run("建部门 无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := postBodyAuth(t, r, "/api/v1/departments", `{"legalEntityId":1,"name":"财务部"}`, token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("建部门 成功返回 id", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := postBodyAuth(t, r, "/api/v1/departments", `{"legalEntityId":1,"name":"财务部"}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
			Data struct {
				ID int64 `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code != 0 || env.Data.ID != 1 {
			t.Fatalf("env=%+v", env)
		}
	})

	t.Run("改部门 成功", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := putAuth(t, r, "/api/v1/departments/1", `{"legalEntityId":2,"name":"客服部"}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("建岗位 无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := postBodyAuth(t, r, "/api/v1/posts", `{"deptId":1,"code":"agent","name":"客服坐席"}`, token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("建岗位 带 roles 成功", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := postBodyAuth(t, r, "/api/v1/posts",
			`{"deptId":1,"code":"agent","name":"客服坐席","roles":["ops"]}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("改岗位 成功", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := putAuth(t, r, "/api/v1/posts/1",
			`{"deptId":2,"code":"dispatcher","name":"装维调度员","roles":["technician"]}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("建岗位 缺参 参数错误", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: true}, mgr)
		w := postBodyAuth(t, r, "/api/v1/posts", `{"deptId":1}`, token)
		var env struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Code == 0 {
			t.Fatalf("expect non-zero code, body=%s", w.Body.String())
		}
	})
}
