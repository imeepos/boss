package adminapi

// 契约:GET/PUT /sms-config 与 POST /sms-config/channel/test(menu:smsconfig)。
// secret 字段:落库密文(enc:v1:)、GET 掩码、PUT 空串跳过。

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestSMSConfigRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("GET 空配置回默认值,secret 掩码", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := getJSON(t, r, "/api/admin/v1/sms-config", token)
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
		if env.Data.Fields["sms.provider"].Value != "aliyun_intl" {
			t.Fatalf("default missing: %+v", env.Data.Fields)
		}
		if env.Data.Fields["sms.accessKeySecret"].Value != "" || env.Data.Fields["sms.accessKeySecret"].HasValue {
			t.Fatalf("secret should be masked-empty")
		}
	})

	t.Run("PUT channel secret 加密落库 空串跳过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}}
		r := newAuthTestRouter(f, mgr)
		w := putAuth(t, r, "/api/admin/v1/sms-config/channel",
			`{"values":{"sms.accessKeyId":"LTAI123","sms.accessKeySecret":"s3cret"}}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		stored := f.params["sms.accessKeySecret"]
		if !strings.HasPrefix(stored, "enc:v1:") || strings.Contains(stored, "s3cret") {
			t.Fatalf("secret not encrypted: %q", stored)
		}
		w2 := putAuth(t, r, "/api/admin/v1/sms-config/channel",
			`{"values":{"sms.accessKeyId":"LTAI456","sms.accessKeySecret":""}}`, token)
		if w2.Code != 200 {
			t.Fatalf("status=%d body=%s", w2.Code, w.Body.String())
		}
		if f.params["sms.accessKeySecret"] != stored || f.params["sms.accessKeyId"] != "LTAI456" {
			t.Fatalf("empty secret should be skipped, from updated")
		}
	})

	t.Run("PUT 越组 key 拒绝", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := putAuth(t, r, "/api/admin/v1/sms-config/template",
			`{"values":{"sms.accessKeyId":"x"}}`, token)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"code":42200`) {
			t.Fatalf("cross-group key accepted: %d %s", w.Code, w.Body.String())
		}
	})

	t.Run("test 自检 缺必填报错 草稿补齐通过 未启用跳过", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{}}
		r := newAuthTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/sms-config/channel/test", `{}`, token)
		if !strings.Contains(w.Body.String(), `"ok":false`) {
			t.Fatalf("missing fields should fail: %s", w.Body.String())
		}
		w2 := postJSONAuth(t, r, "/api/admin/v1/sms-config/channel/test",
			`{"values":{"sms.accessKeyId":"k","sms.accessKeySecret":"s","sms.contentCode.cn":"c1"}}`, token)
		if !strings.Contains(w2.Body.String(), `"ok":true`) {
			t.Fatalf("draft test should pass: %s", w2.Body.String())
		}
		w3 := putAuth(t, r, "/api/admin/v1/sms-config/channel", `{"values":{"sms.enabled":"false"}}`, token)
		if w3.Code != 200 {
			t.Fatalf("disable failed")
		}
		w4 := postJSONAuth(t, r, "/api/admin/v1/sms-config/channel/test", `{}`, token)
		if !strings.Contains(w4.Body.String(), "跳过自检") {
			t.Fatalf("disabled should skip: %s", w4.Body.String())
		}
	})

	t.Run("test 不支持区号报错", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}, params: map[string]string{
			"sms.accessKeyId": "k", "sms.accessKeySecret": "s",
		}}
		r := newAuthTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/sms-config/channel/test",
			`{"phone":"+19995551234"}`, token)
		if !strings.Contains(w.Body.String(), "区号不支持") {
			t.Fatalf("unsupported region should fail: %s", w.Body.String())
		}
	})
}
