package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(Config{CORSOrigins: []string{"http://localhost:5174"}})
	r.POST("/api/admin/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/api/admin/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5174")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5174" {
		t.Fatalf("allow origin = %q", got)
	}
}

func TestCORSAllowAnyOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(Config{})

	req := httptest.NewRequest(http.MethodOptions, "/healthz", nil)
	req.Header.Set("Origin", "http://evil.example")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://evil.example" {
		t.Fatalf("allow origin = %q, want echo", got)
	}
}

func TestCORSLocalhostAnyPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(Config{CORSOrigins: []string{"http://localhost:5174"}})

	for _, origin := range []string{"http://localhost:5175", "http://127.0.0.1:9999", "http://localhost:5173"} {
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		req.Header.Set("Origin", origin)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if got := res.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Fatalf("allow origin = %q, want %q", got, origin)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	req.Header.Set("Origin", "http://192.168.0.5:5173")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://192.168.0.5:5173" {
		t.Fatalf("non-local ip should be echoed, got %q", got)
	}
}
