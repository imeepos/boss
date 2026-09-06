package adminapi

// A1 契约:POST /lo-accounts/{loid}/reset-password(menu:loaccount)——
// 随机密码一次性明文返回;LOID 不存在 404;无权限 403。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type resetStubAaa struct {
	fakeAaa
	loid     string
	resetErr error
}

func (s *resetStubAaa) ResetLoPassword(_ context.Context, loid string) (string, error) {
	s.loid = loid
	return "Rand0m-Pw-x8k2Qm", s.resetErr
}

func newAaaCredRouter(aa aaa.AaaService, permOk bool, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: &fakeUser{permOk: permOk}, Aaa: aa}, mgr)
	return r
}

func TestLoAccountResetPassword(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("无权限 403", func(t *testing.T) {
		r := newAaaCredRouter(&resetStubAaa{}, false, mgr)
		w := postAuth(t, r, "/api/admin/v1/lo-accounts/LOID-1/reset-password", token)
		if w.Code != http.StatusForbidden {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("重置成功 一次性明文返回", func(t *testing.T) {
		stub := &resetStubAaa{}
		r := newAaaCredRouter(stub, true, mgr)
		w := postAuth(t, r, "/api/admin/v1/lo-accounts/LOID-77/reset-password", token)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Code int `json:"code"`
			Data struct {
				Loid     string `json:"loid"`
				Password string `json:"password"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Code != 0 || env.Data.Loid != "LOID-77" || env.Data.Password == "" {
			t.Fatalf("env=%+v", env)
		}
		if stub.loid != "LOID-77" {
			t.Fatalf("captured loid=%q", stub.loid)
		}
	})

	t.Run("LOID 不存在 envelope 40400", func(t *testing.T) {
		stub := &resetStubAaa{resetErr: aaa.ErrNotFound}
		r := newAaaCredRouter(stub, true, mgr)
		w := postAuth(t, r, "/api/admin/v1/lo-accounts/LOID-MISS/reset-password", token)
		var env struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Code != 40400 { // envelope 业务码:资源不存在(apitypes.CodeNotFound)
			t.Fatalf("code=%d want 40400 body=%s", env.Code, w.Body.String())
		}
		if stub.loid != "LOID-MISS" {
			t.Fatalf("captured loid=%q", stub.loid)
		}
	})

	t.Run("服务端错误透传 envelope 50000", func(t *testing.T) {
		stub := &resetStubAaa{resetErr: errors.New("pg down")}
		r := newAaaCredRouter(stub, true, mgr)
		w := postAuth(t, r, "/api/admin/v1/lo-accounts/LOID-1/reset-password", token)
		var env struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Code != 50000 { // envelope 业务码:未知错误一律 50000(apitypes.CodeInternal)
			t.Fatalf("code=%d want 50000 body=%s", env.Code, w.Body.String())
		}
	})
}
