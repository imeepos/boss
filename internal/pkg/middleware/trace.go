package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// CtxTraceID trace-id 在 request context 中的键,供日志/APM 贯穿整条链路。
const CtxTraceID = "boss.trace_id"

// TraceID 注入:为每个请求生成/透传 trace-id 写入响应头,并开启 OTel server span。
// 上游若已带 X-Request-Id 则透传,保证跨服务串联;span 携带同一 id 供 Jaeger 检索。
func TraceID() gin.HandlerFunc {
	tracer := otel.Tracer("boss/http")
	return func(c *gin.Context) {
		tid := c.GetHeader("X-Request-Id")
		if tid == "" {
			tid = newTraceID()
		}
		c.Set(CtxTraceID, tid)
		c.Header("X-Request-Id", tid)

		ctx, span := tracer.Start(c.Request.Context(), c.Request.Method+" "+c.FullPath(),
			trace.WithAttributes(
				attribute.String("boss.trace_id", tid),
				attribute.String("http.request.method", c.Request.Method),
				semconv.URLPath(c.Request.URL.Path),
			))
		c.Request = c.Request.WithContext(ctx)
		defer span.End()
		c.Next()
		span.SetAttributes(semconv.HTTPResponseStatusCode(c.Writer.Status()))
	}
}

func newTraceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
