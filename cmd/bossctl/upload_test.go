package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestUploadMultipart 校验 upload 以 multipart 字段 file 上传并解统一信封。
func TestUploadMultipart(t *testing.T) {
	var gotFormFile, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/worker/v1/attachments/upload" {
			t.Errorf("路径 = %s,期望 /api/worker/v1/attachments/upload", r.URL.Path)
		}
		gotAuth = r.Header.Get("X-API-Key")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		fh, err := r.MultipartForm.File["file"][0].Open()
		if err != nil {
			t.Fatalf("打开 file 字段: %v", err)
		}
		defer fh.Close()
		buf := make([]byte, 32)
		n, _ := fh.Read(buf)
		gotFormFile = string(buf[:n])
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "success", "data": map[string]any{"objectKey": "att/1"}})
	}))
	defer srv.Close()

	c := &CLI{cfg: &config{Server: srv.URL, APIKey: "boss_test"}}
	if err := c.upload([]string{"--portal", "worker", "testdata/upload.txt"}); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if gotAuth != "boss_test" {
		t.Errorf("X-API-Key = %q,期望 boss_test", gotAuth)
	}
	if gotFormFile != "cli-upload-test" {
		t.Errorf("file 内容 = %q,期望 cli-upload-test", gotFormFile)
	}
}

// TestUploadBizError 业务错误码必须返回 error(退出码 1,CI 可感知失败)。
func TestUploadBizError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"code": 40100, "msg": "invalid api key"})
	}))
	defer srv.Close()
	c := &CLI{cfg: &config{Server: srv.URL}}
	if err := c.upload([]string{"testdata/upload.txt"}); err == nil {
		t.Fatal("upload 业务错误应返回 error(退出码 1),实际 nil")
	}
}

// TestResolvePath 校验三端路径补全(user 端真实前缀是 /api/user/v1,不是 /api/v1)。
func TestResolvePath(t *testing.T) {
	cases := map[string]string{
		"/orders":                "/api/admin/v1/orders",
		"user:/orders":           "/api/user/v1/orders",
		"worker:/home":           "/api/worker/v1/home",
		"/api/user/v1/orders":    "/api/user/v1/orders",
		"/api/worker/v1/tickets": "/api/worker/v1/tickets",
		"/api/admin/v1/accounts": "/api/admin/v1/accounts",
	}
	for in, want := range cases {
		if got := resolvePath(in); got != want {
			t.Errorf("resolvePath(%q) = %q,期望 %q", in, got, want)
		}
	}
}
