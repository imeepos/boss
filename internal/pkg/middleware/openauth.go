// 开放平台中间件:对外部集成方的 /api/open/v1 请求做 HMAC 验签 + 限流 + 配额。
//
// 链路:OpenAuth(验签+配额计数) → RateLimit(每应用 RPM 令牌桶) → 业务 handler。
// 当前部署要求单实例;进程内 RPM 令牌桶仅为短窗口保护，日配额仍以 DB 为准。
// 扩容前必须替换为共享限流存储，避免每副本放大 RPM。
package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/openplat"
)

// CtxOpenApp 验签通过的应用行 ID 上下文键。
const CtxOpenApp = "boss.openapp"

// CtxOpenSandbox 沙箱应用标记上下文键。
const CtxOpenSandbox = "boss.opensandbox"

// OpenAuth 开放平台验签中间件:
// 校验 X-BOSS-AppId/Timestamp/Nonce/Signature,通过后记用量并检查日配额。
func OpenAuth(svc openplat.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader("X-BOSS-AppId")
		ts := c.GetHeader("X-BOSS-Timestamp")
		nonce := c.GetHeader("X-BOSS-Nonce")
		sig := c.GetHeader("X-BOSS-Signature")
		if appID == "" || ts == "" || nonce == "" || sig == "" {
			abortOpen(c, http.StatusUnauthorized, "missing signature headers")
			return
		}
		auth, err := svc.LookupActive(c.Request.Context(), appID)
		if err != nil {
			abortOpen(c, http.StatusUnauthorized, "unknown or disabled app")
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		_ = c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		if err := openplat.VerifyRequest(auth, c.Request, body, appID, ts, nonce, sig); err != nil {
			abortOpen(c, http.StatusUnauthorized, "signature verification failed")
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		count, err := svc.TouchUsage(c.Request.Context(), auth.AppRowID, today)
		if err != nil {
			abortOpen(c, http.StatusInternalServerError, "usage accounting failed")
			return
		}
		if auth.DailyQuota > 0 && count > int64(auth.DailyQuota) {
			c.Header("X-BOSS-Quota-Exceeded", "true")
			abortOpen(c, http.StatusTooManyRequests, "daily quota exceeded")
			return
		}
		c.Set(CtxOpenApp, auth.AppRowID)
		c.Set(CtxOpenRPM, auth.RateLimitRPM)
		c.Set(CtxOpenSandbox, auth.Sandbox)
		c.Next()
	}
}

// abortOpen 开放面统一错误响应(与三端 {code,msg} 形状一致,面向集成方)。
func abortOpen(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"code": status, "msg": msg})
}

// OpenAppIDFrom 取验签通过的应用行 ID;未通过验签返回 0。
func OpenAppIDFrom(c *gin.Context) int64 {
	if v, ok := c.Get(CtxOpenApp); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// OpenSandboxFrom 判定请求来自沙箱应用(M4:开放面只见样例数据)。
func OpenSandboxFrom(c *gin.Context) bool {
	v, _ := c.Get(CtxOpenSandbox)
	sb, _ := v.(bool)
	return sb
}
