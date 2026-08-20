package adminapi

// 契约:GET/PUT /realid-config 与 POST /realid-config/channel/test(menu:realidconfig)。
// secret 字段:落库密文(enc:v1:)、GET 掩码、PUT 空串跳过。

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestRealIDConfigRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("GET 空配置回默认值,secret 掩码", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := getJSON(t, r, "/api/admin/v1/realid-config", token)
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
		if env.Data.Fields["realid.provider"].Value != "aliyun_cloudauth" {
			t.Fatalf("default missing: %+v", env.Data.Fields)
		}
		if env.Data.Fields["realid.accessKeySecret"].Value != "" || env.Data.Fields["realid.accessKeySecret"].HasValue {
			t.Fatalf("secret should be masked-empty")
		}
	})

	t.Run("PUT channel secret 加密落库 空串跳过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}}
		r := newAuthTestRouter(f, mgr)
		w := putAuth(t, r, "/api/admin/v1/realid-config/channel",
			`{"values":{"realid.accessKeyId":"LTAI123","realid.accessKeySecret":"s3cret"}}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		stored := f.params["realid.accessKeySecret"]
		if !strings.HasPrefix(stored, "enc:v1:") || strings.Contains(stored, "s3cret") {
			t.Fatalf("secret not encrypted: %q", stored)
		}
		w2 := putAuth(t, r, "/api/admin/v1/realid-config/channel",
			`{"values":{"realid.endpoint":"https://cloudauth.example.com/","realid.accessKeySecret":""}}`, token)
		if w2.Code != 200 {
			t.Fatalf("status=%d body=%s", w2.Code, w2.Body.String())
		}
		if f.params["realid.accessKeySecret"] != stored || f.params["realid.endpoint"] == "" {
			t.Fatalf("empty secret should be skipped, endpoint updated")
		}
	})

	t.Run("PUT 越组 key 拒绝", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := putAuth(t, r, "/api/admin/v1/realid-config/other",
			`{"values":{"realid.accessKeyId":"x"}}`, token)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":42200`) {
			t.Fatalf("unknown group accepted: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("test 自检 缺必填报错 草稿补齐通过 未启用跳过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{}}
		r := newAuthTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/realid-config/channel/test", `{}`, token)
		if !strings.Contains(w.Body.String(), `"ok":false`) {
			t.Fatalf("missing fields should fail: %s", w.Body.String())
		}
		w2 := postJSONAuth(t, r, "/api/admin/v1/realid-config/channel/test",
			`{"values":{"realid.accessKeyId":"k","realid.accessKeySecret":"s"}}`, token)
		if !strings.Contains(w2.Body.String(), `"ok":true`) {
			t.Fatalf("draft test should pass: %s", w2.Body.String())
		}
		w3 := putAuth(t, r, "/api/admin/v1/realid-config/channel", `{"values":{"realid.enabled":"false"}}`, token)
		if w3.Code != 200 {
			t.Fatalf("disable failed")
		}
		w4 := postJSONAuth(t, r, "/api/admin/v1/realid-config/channel/test", `{}`, token)
		if !strings.Contains(w4.Body.String(), "人工核验") {
			t.Fatalf("disabled should skip: %s", w4.Body.String())
		}
	})
}
