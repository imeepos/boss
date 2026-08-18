package server

// W11 可观测单测:/metrics 暴露、业务端点计入 RED 指标、基础设施端点不计入。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := New(Config{})
	r.GET("/api/v1/ping/:id", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	do := func(path string) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s code=%d", path, w.Code)
		}
	}
	do("/healthz")
	do("/api/v1/ping/42") // 参数路由,应折叠为模板路径
	do("/metrics")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := w.Body.String()
	if !strings.Contains(body, "boss_http_requests_total") ||
		!strings.Contains(body, `path="/api/v1/ping/:id"`) ||
		!strings.Contains(body, "boss_http_request_duration_seconds") {
		t.Fatalf("metrics missing RED series:\n%s", body)
	}
	// healthz 自身不进业务指标。
	if strings.Contains(body, `path="/healthz"`) {
		t.Fatal("healthz should be excluded from request metrics")
	}
	if !strings.Contains(body, "go_goroutines") {
		t.Fatal("runtime metrics missing")
	}
}
