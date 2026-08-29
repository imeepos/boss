package apiclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallEnvelopeAndAuthHeader(t *testing.T) {
	var gotKey, gotPath, gotQuery, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": map[string]any{"id": 1}})
	}))
	defer srv.Close()

	c := New(srv.URL, "boss_userkey")
	res, err := c.Call("POST", "/api/user/v1/orders", map[string]any{"offerId": 1}, map[string]string{"page": "1", "kw": "宽带 安装"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if gotKey != "boss_userkey" {
		t.Errorf("X-API-Key = %q", gotKey)
	}
	if gotPath != "/api/user/v1/orders" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "kw=%E5%AE%BD%E5%B8%A6+%E5%AE%89%E8%A3%85&page=1" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotBody != `{"offerId":1}` {
		t.Errorf("body = %q", gotBody)
	}
	if res.Envelope == nil || res.Envelope.Code != 0 || res.StatusCode != http.StatusOK {
		t.Errorf("envelope = %+v status = %d", res.Envelope, res.StatusCode)
	}
}

func TestCallBusinessErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 40300, "msg": "无权限"})
	}))
	defer srv.Close()

	res, err := New(srv.URL, "k").Call("GET", "/x", nil, nil)
	if err != nil {
		t.Fatalf("transport error not expected: %v", err)
	}
	if res.Envelope == nil || res.Envelope.Code != 40300 || res.Envelope.Msg != "无权限" {
		t.Errorf("envelope = %+v", res.Envelope)
	}
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d", res.StatusCode)
	}
}

func TestCallNonJSONFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 binary"))
	}))
	defer srv.Close()

	res, err := New(srv.URL, "k").Call("GET", "/receipt.pdf", nil, nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if res.Envelope != nil {
		t.Errorf("envelope should be nil for pdf, got %+v", res.Envelope)
	}
	if !strings.HasPrefix(string(res.Body), "%PDF") {
		t.Errorf("body = %q", string(res.Body))
	}
}

func TestCallTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // 立即关闭制造连接失败
	if _, err := New(srv.URL, "k").Call("GET", "/x", nil, nil); err == nil {
		t.Fatal("expected transport error")
	}
}

func TestNewTrimsTrailingSlashAndDefaultServer(t *testing.T) {
	if c := New("http://x:1/", "k"); c.Server != "http://x:1" {
		t.Errorf("server = %q", c.Server)
	}
	if c := New("", "k"); c.Server != DefaultServer {
		t.Errorf("default server = %q", c.Server)
	}
}
