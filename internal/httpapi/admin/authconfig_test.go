package adminapi

// 契约:GET/PUT /auth-config 与 POST /auth-config/{group}/test(menu:authconfig)。
// secret 字段:落库密文(enc:v1:)、GET 掩码、PUT 空串跳过。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

type fakeAuthUser struct {
	fakeUser
	params map[string]string
}

func (f *fakeAuthUser) ListParams(context.Context) ([]user.Param, error) {
	out := make([]user.Param, 0, len(f.params))
	for k, v := range f.params {
		out = append(out, user.Param{Key: k, Value: v})
	}
	return out, nil
}

func (f *fakeAuthUser) UpdateParam(_ context.Context, key, value string, _ int64) error {
	if f.params == nil {
		f.params = map[string]string{}
	}
	f.params[key] = value
	return nil
}

func newAuthTestRouter(f *fakeAuthUser, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: f}, mgr)
	return r
}

func TestAuthConfigRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("无权限 403", func(t *testing.T) {
		r := newTestRouter(&fakeUser{permOk: false}, mgr)
		w := getJSON(t, r, "/api/admin/v1/auth-config", token)
		if w.Code != 403 {
			t.Fatalf("status=%d want 403", w.Code)
		}
	})

	t.Run("GET 空配置回默认值", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := getJSON(t, r, "/api/admin/v1/auth-config", token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var env struct {
			Data struct {
				Fields map[string]struct {
					Value    string `json:"value"`
					HasValue bool   `json:"hasValue"`
				} `json:"fields"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if env.Data.Fields["auth.cn.preloadTimeoutMs"].Value != "5000" {
			t.Fatalf("default missing: %+v", env.Data.Fields)
		}
		if env.Data.Fields["auth.cn.appSecret"].Value != "" || env.Data.Fields["auth.cn.appSecret"].HasValue {
			t.Fatalf("secret should be masked-empty")
		}
	})

	t.Run("PUT cn secret 加密落库 空串跳过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}}
		r := newAuthTestRouter(f, mgr)
		w := putAuth(t, r, "/api/admin/v1/auth-config/cn",
			`{"values":{"auth.cn.enabled":"true","auth.cn.appKey":"jk-123","auth.cn.appSecret":"s3cret","auth.cn.packageName":"com.ymm.boss.user"}}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		stored := f.params["auth.cn.appSecret"]
		if !strings.HasPrefix(stored, "enc:v1:") || strings.Contains(stored, "s3cret") {
			t.Fatalf("secret not encrypted: %q", stored)
		}
		// 二次 PUT:appSecret 空串 → 不修改,appKey 更新
		w2 := putAuth(t, r, "/api/admin/v1/auth-config/cn",
			`{"values":{"auth.cn.appKey":"jk-456","auth.cn.appSecret":""}}`, token)
		if w2.Code != 200 {
			t.Fatalf("status=%d body=%s", w2.Code, w2.Body.String())
		}
		if f.params["auth.cn.appSecret"] != stored {
			t.Fatalf("empty secret should be skipped")
		}
		if f.params["auth.cn.appKey"] != "jk-456" {
			t.Fatalf("appKey not updated")
		}
	})

	t.Run("PUT 越组 key 拒绝", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := putAuth(t, r, "/api/admin/v1/auth-config/my",
			`{"values":{"auth.cn.appKey":"x"}}`, token)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":42200`) {
			t.Fatalf("cross-group key accepted: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("test 自检 缺必填报错 补齐通过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{
			"auth.cn.enabled": "true", "auth.cn.appKey": "k",
		}}
		r := newAuthTestRouter(f, mgr)
		body := `{"values":{"auth.cn.appSecret":"draft-secret","auth.cn.packageName":"com.x"}}`
		w := postJSONAuth(t, r, "/api/admin/v1/auth-config/cn/test", body, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"ok":true`) {
			t.Fatalf("draft test should pass: %s", w.Body.String())
		}
		// 无草稿:缺 appSecret/packageName
		w2 := postJSONAuth(t, r, "/api/admin/v1/auth-config/cn/test", `{}`, token)
		if !strings.Contains(w2.Body.String(), `"ok":false`) {
			t.Fatalf("missing fields should fail: %s", w2.Body.String())
		}
	})
}
