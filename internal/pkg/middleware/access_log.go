package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog 结构化访问日志:method/path/status/duration/trace_id。
// trace_id 从 TraceID 中间件写入的 context 键取;未注入则为空。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		tid, _ := c.Get(CtxTraceID)
		log.Printf("access method=%s path=%s status=%d dur=%s trace_id=%v",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start), tid)
	}
}
