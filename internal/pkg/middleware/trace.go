package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// CtxTraceID trace-id 在 request context 中的键,供日志/APM 贯穿整条链路。
const CtxTraceID = "boss.trace_id"

// TraceID 注入:为每个请求生成/透传 trace-id,并写入响应头,便于日志与链路追踪关联。
// 上游若已带 X-Request-Id 则透传,否则生成,保证跨服务可串联(发现 2.3 可观测契约)。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.GetHeader("X-Request-Id")
		if tid == "" {
			tid = newTraceID()
		}
		c.Set(CtxTraceID, tid)
		c.Header("X-Request-Id", tid)
		c.Next()
	}
}

func newTraceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
