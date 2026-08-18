package middleware

// trace 中间件 × OTel 单测:请求生成 span,记录 http 属性与 boss.trace_id。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTraceIDStartsSpan(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(TraceID())
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-Id", "tid-42")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Header().Get("X-Request-Id") != "tid-42" {
		t.Fatalf("code=%d header=%q", w.Code, w.Header().Get("X-Request-Id"))
	}
	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans=%d want 1", len(spans))
	}
	s := spans[0]
	if s.Name() != "GET /ping" {
		t.Fatalf("span name=%q", s.Name())
	}
	attrs := map[string]string{}
	for _, a := range s.Attributes() {
		attrs[string(a.Key)] = a.Value.Emit()
	}
	if attrs["boss.trace_id"] != "tid-42" || attrs["http.request.method"] != "GET" {
		t.Fatalf("attrs=%v", attrs)
	}
}
