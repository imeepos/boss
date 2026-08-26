// 开放平台每应用 RPM 令牌桶(进程内存);日配额由 open_usage_day 持久化。
// 当前部署要求单实例；扩容前必须替换为共享限流存储。
package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// openBucket 单应用令牌桶。
type openBucket struct {
	tokens float64
	last   time.Time
}

// OpenRateLimiter 进程内限流器;按应用行 ID 隔离。
type OpenRateLimiter struct {
	mu      sync.Mutex
	buckets map[int64]*openBucket
}

// NewOpenRateLimiter 构造限流器。
func NewOpenRateLimiter() *OpenRateLimiter {
	return &OpenRateLimiter{buckets: map[int64]*openBucket{}}
}

// OpenRateLimit 返回限流中间件:超过应用的 RPM 上限返回 429。
// 速率取值在 OpenAuth 之后(依赖应用上下文),rpm<=0 视为不限。
func (l *OpenRateLimiter) OpenRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		appRow := OpenAppIDFrom(c)
		if appRow == 0 {
			c.Next()
			return
		}
		rpm := openRPMFrom(c)
		if rpm <= 0 {
			c.Next()
			return
		}
		if !l.allow(appRow, rpm) {
			c.Header("X-BOSS-RateLimit-Rpm", strconv.Itoa(rpm))
			c.Header("Retry-After", "1")
			abortOpen(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		c.Next()
	}
}

// allow 按令牌桶判定;容量=rate,以 rate/秒 补充。
func (l *OpenRateLimiter) allow(id int64, rpm int) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[id]
	if !ok {
		b = &openBucket{tokens: float64(rpm), last: now}
		l.buckets[id] = b
	}
	rate := float64(rpm) / 60.0
	b.tokens += now.Sub(b.last).Seconds() * rate
	if b.tokens > float64(rpm) {
		b.tokens = float64(rpm)
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// openRPMFrom 从上下文读取应用 RPM(由 OpenAuth 验签时注入)。
func openRPMFrom(c *gin.Context) int {
	if v, ok := c.Get(CtxOpenRPM); ok {
		if n, ok := v.(int); ok {
			return n
		}
	}
	return 0
}

// CtxOpenRPM 应用 RPM 上下文键(装配层注入)。
const CtxOpenRPM = "boss.openrpm"
