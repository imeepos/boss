package app

// 契约:POST /auth/change-password —— 自助改密;成功 200/旧口令错 401/新口令过短 400。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// envCode 解响应 envelope 业务码(HTTP 恒 200,错误经 code 表达)。
func envCode(t *testing.T, w *httptest.ResponseRecorder) apitypes.Code {
	t.Helper()
	var env struct {
		Code apitypes.Code `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("bad envelope: %v body=%s", err, w.Body.String())
	}
	return apitypes.Code(env.Code)
}

func postJSONAuth(t *testing.T, r *gin.Engine, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestChangePasswordHandler(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(1, "boss", "sysadmin")
	body := `{"oldPassword":"old123","newPassword":"new123456"}`

	t.Run("成功", func(t *testing.T) {
		r := newTestRouter(&fakeUser{}, mgr)
		if w := postJSONAuth(t, r, "/api/v1/auth/change-password", body, token); envCode(t, w) != apitypes.CodeOK {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("旧口令错误未授权", func(t *testing.T) {
		f := &fakeUser{changeErr: user.ErrUnauthorized}
		r := newTestRouter(f, mgr)
		if w := postJSONAuth(t, r, "/api/v1/auth/change-password", body, token); envCode(t, w) != apitypes.CodeUnauthorized {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("新口令过短参数非法", func(t *testing.T) {
		r := newTestRouter(&fakeUser{}, mgr)
		short := `{"oldPassword":"old123","newPassword":"abc"}`
		if w := postJSONAuth(t, r, "/api/v1/auth/change-password", short, token); envCode(t, w) != apitypes.CodeInvalidParam {
			t.Fatalf("body=%s", w.Body.String())
		}
	})

	t.Run("未认证 401", func(t *testing.T) {
		r := newTestRouter(&fakeUser{}, mgr)
		if w := postJSONAuth(t, r, "/api/v1/auth/change-password", body, "bad-token"); w.Code != 401 {
			t.Fatalf("status=%d", w.Code)
		}
	})
}
