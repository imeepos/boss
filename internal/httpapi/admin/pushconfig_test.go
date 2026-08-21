package adminapi

// 契约:GET/PUT /push-config 与 POST /push-config/channel/test(menu:pushconfig)。
// secret 字段:落库密文(enc:v1:)、GET 掩码、PUT 空串跳过。

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

func TestPushConfigRoutes(t *testing.T) {
	mgr := auth.NewManager("test-secret", time.Hour)
	token, _ := mgr.Sign(auth.AudAdmin, 1, "boss", "sysadmin")

	t.Run("GET 空配置回默认值,secret 掩码", func(t *testing.T) {
		r := newAuthTestRouter(&fakeAuthUser{fakeUser: fakeUser{permOk: true}}, mgr)
		w := getJSON(t, r, "/api/admin/v1/push-config", token)
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
		f := env.Data.Fields
		if f["push.provider"].Value != "jpush" || f["push.jpush.liveTime"].Value != "86400" {
			t.Fatalf("defaults missing: %+v", f)
		}
		if !strings.HasPrefix(f["push.jpush.apiUrl"].Value, "https://") {
			t.Fatalf("apiURL default missing: %+v", f)
		}
		if f["push.jpush.masterSecret"].Value != "" || f["push.jpush.masterSecret"].HasValue {
			t.Fatalf("secret should be masked-empty")
		}
	})

	t.Run("PUT channel secret 加密落库 空串跳过 越组拒绝", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}}
		r := newAuthTestRouter(f, mgr)
		w := putAuth(t, r, "/api/admin/v1/push-config/channel",
			`{"values":{"push.jpush.appKey":"ak123","push.jpush.masterSecret":"ms3cret"}}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		stored := f.params["push.jpush.masterSecret"]
		if !strings.HasPrefix(stored, "enc:v1:") || strings.Contains(stored, "ms3cret") {
			t.Fatalf("secret not encrypted: %q", stored)
		}
		w2 := putAuth(t, r, "/api/admin/v1/push-config/channel",
			`{"values":{"push.jpush.appKey":"ak456","push.jpush.masterSecret":""}}`, token)
		if w2.Code != 200 {
			t.Fatalf("status=%d body=%s", w2.Code, w2.Body.String())
		}
		if f.params["push.jpush.masterSecret"] != stored || f.params["push.jpush.appKey"] != "ak456" {
			t.Fatalf("empty secret should be skipped, appKey updated")
		}
		w3 := putAuth(t, r, "/api/admin/v1/push-config/channel",
			`{"values":{"sms.from":"x"}}`, token)
		if w3.Code != 200 {
			t.Fatalf("cross-group key should be rejected, got %d", w3.Code)
		}
		w4 := putAuth(t, r, "/api/admin/v1/push-config/none", `{"values":{}}`, token)
		if w4.Code != 200 {
			t.Fatalf("unknown group should be rejected, got %d", w4.Code)
		}
	})

	t.Run("POST test 完整性校验", func(t *testing.T) {
		f := &fakeAuthUser{fakeUser: fakeUser{permOk: true}}
		r := newAuthTestRouter(f, mgr)
		w := postJSONAuth(t, r, "/api/admin/v1/push-config/channel/test",
			`{"values":{}}`, token)
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "缺少必填字段") {
			t.Fatalf("completeness should report missing: %s", w.Body.String())
		}
		// 未启用视为通过。
		w2 := postJSONAuth(t, r, "/api/admin/v1/push-config/channel/test",
			`{"values":{"push.enabled":"false"}}`, token)
		if !strings.Contains(w2.Body.String(), "跳过自检") {
			t.Fatalf("disabled should pass: %s", w2.Body.String())
		}
	})
}
