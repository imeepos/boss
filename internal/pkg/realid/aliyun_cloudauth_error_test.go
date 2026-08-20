package realid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAliyunCloudauth_DefaultEndpoint 空 Endpoint 构造期落 DefaultEndpoint(不发请求)。
func TestAliyunCloudauth_DefaultEndpoint(t *testing.T) {
	v := NewAliyunCloudauth(AliyunCloudauthConfig{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if v == nil {
		t.Fatal("configured channel must not be nil")
	}
	s := v.(*aliyunCloudauth)
	if s.endpoint != DefaultEndpoint {
		t.Fatalf("endpoint=%s want %s", s.endpoint, DefaultEndpoint)
	}
}

// TestAliyunCloudauth_TransportError 服务不可达 → call 报错。
func TestAliyunCloudauth_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // 立即关闭,连接必失败
	v := NewAliyunCloudauth(AliyunCloudauthConfig{AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: url})
	if _, err := v.Verify(context.Background(), "张三", "x"); err == nil {
		t.Fatal("expect transport error")
	}
}

// TestAliyunCloudauth_BadJSON 非法响应体 → bad response 错误。
func TestAliyunCloudauth_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`not-json`))
	}))
	defer srv.Close()
	v := NewAliyunCloudauth(AliyunCloudauthConfig{AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: srv.URL})
	if _, err := v.Verify(context.Background(), "张三", "x"); err == nil {
		t.Fatal("expect bad response error")
	}
}

// TestAliyunCloudauth_UnknownBizCode 缺 ResultObject/BizCode → unknown BizCode 错误。
func TestAliyunCloudauth_UnknownBizCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"Code":"200"}`))
	}))
	defer srv.Close()
	v := NewAliyunCloudauth(AliyunCloudauthConfig{AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: srv.URL})
	if _, err := v.Verify(context.Background(), "张三", "x"); err == nil {
		t.Fatal("expect unknown BizCode error")
	}
}

// TestAliyunCloudauth_BadEndpointURL 非法 Endpoint → NewRequest 构造失败。
func TestAliyunCloudauth_BadEndpointURL(t *testing.T) {
	v := NewAliyunCloudauth(AliyunCloudauthConfig{
		AccessKeyID: "ak", AccessKeySecret: "sk", Endpoint: "http://127.0.0.1:1/\x7f",
	})
	if _, err := v.Verify(context.Background(), "张三", "x"); err == nil {
		t.Fatal("expect request build error")
	}
}
