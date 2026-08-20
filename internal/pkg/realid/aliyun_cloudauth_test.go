package realid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestAliyunCloudauth_BizCode 官方契约:Code=200 + BizCode 1/2/3 → PASS/FAIL/FAIL。
func TestAliyunCloudauth_BizCode(t *testing.T) {
	cases := []struct {
		biz  string
		want string
	}{
		{"1", Pass},
		{"2", Fail},
		{"3", Fail},
	}
	for _, tc := range cases {
		var gotPath string
		var gotForm url.Values
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_ = r.ParseForm()
			gotForm = r.PostForm
			w.Write([]byte(`{"Code":"200","Message":"success","ResultObject":{"BizCode":"` + tc.biz + `"}}`))
		}))
		v := NewAliyunCloudauth(AliyunCloudauthConfig{
			AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: srv.URL,
		})
		if v == nil {
			t.Fatal("configured channel must not be nil")
		}
		got, err := v.Verify(context.Background(), "张三", "110101199001011234")
		if err != nil {
			t.Fatalf("biz=%s: %v", tc.biz, err)
		}
		if got != tc.want {
			t.Fatalf("biz=%s: got %s want %s", tc.biz, got, tc.want)
		}
		if gotPath != "/" {
			t.Fatalf("path=%s", gotPath)
		}
		// 请求参数契约:RPC 四要素 + 二要素明文入参。
		for k, want := range map[string]string{
			"Action": "Id2MetaVerify", "Version": "2019-03-07", "ParamType": "normal",
			"UserName": "张三", "IdentifyNum": "110101199001011234",
			"SignatureMethod": "HMAC-SHA1", "AccessKeyId": "ak", "Format": "JSON",
		} {
			if gotForm.Get(k) != want {
				t.Fatalf("param %s=%q want %q", k, gotForm.Get(k), want)
			}
		}
		if gotForm.Get("Signature") == "" {
			t.Fatal("missing Signature")
		}
		srv.Close()
	}
}

// TestAliyunCloudauth_DisabledNil 凭据缺失 → 通道未启用(nil)。
func TestAliyunCloudauth_DisabledNil(t *testing.T) {
	if v := NewAliyunCloudauth(AliyunCloudauthConfig{}); v != nil {
		t.Fatal("empty creds must yield nil channel")
	}
}

// TestAliyunCloudauth_ApiError Code!=200 → 错误(非 FAIL,由上层落 PENDING 人工兜底)。
func TestAliyunCloudauth_ApiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"Code":"400","Message":"InvalidParam"}`))
	}))
	defer srv.Close()
	v := NewAliyunCloudauth(AliyunCloudauthConfig{
		AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: srv.URL,
	})
	if _, err := v.Verify(context.Background(), "张三", "x"); err == nil {
		t.Fatal("expect error on Code!=200")
	}
}
