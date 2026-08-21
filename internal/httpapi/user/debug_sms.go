package userapi

// 用户端开发模式专用端点:仅在 BOSS_DEBUG_SMS=1 时挂载,
// 用于 App 端开发模式开关后"获取验证码后立即从后台拉取明文自动填入"。
// 严禁进入生产契约(api/openapi/user*.yaml);只在测试/联调环境装配。

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// debugSmsEnabled 启动时从 BOSS_DEBUG_SMS 环境变量读取(默认 false)。
// true 时挂载 /debug/sms-code 回显端点;严禁线上打开,避免验证码明文外泄。
var debugSmsEnabled = os.Getenv("BOSS_DEBUG_SMS") == "1"

// RegisterDebugRoutes 挂载开发模式专用路由。
// 单一 GET 端点:login/register/reset 公开;verify 场景 JWT 必传,中间件解析 claims。
func RegisterDebugRoutes(pub *gin.RouterGroup, a *app.Application, mgr *auth.Manager) {
	if !debugSmsEnabled {
		return
	}
	pub.GET("/debug/sms-code", debugOptionalAuthn(mgr, auth.AudUser), debugSmsCode(a))
}

// debugOptionalAuthn 兼容中间件:有 Bearer 解析 claims,无/错则放行(handler 内按需拒)。
// 单一端点分场景鉴权的常见做法。
func debugOptionalAuthn(m *auth.Manager, aud string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if strings.HasPrefix(h, "Bearer ") {
			if claims, err := m.Verify(strings.TrimPrefix(h, "Bearer ")); err == nil && claims.Aud == aud {
				c.Set(middleware.CtxClaims, claims)
			}
		}
		c.Next()
	}
}

// debugSmsCode GET /debug/sms-code?phone=&scene=
// GET /debug/sms-code?scene=verify (需 JWT,phone 从 claims 解)
func debugSmsCode(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		scene := c.Query("scene")
		phone := c.Query("phone")
		if scene == "verify" && phone == "" {
			cid, ok := requireCustomer(c)
			if !ok {
				return
			}
			phone = portalCustomerPhone(c.Request.Context(), a, cid)
		}
		if phone == "" || scene == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "phone and scene required"})
			return
		}
		if scene != "login" && scene != "register" && scene != "reset" && scene != "verify" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "invalid scene"})
			return
		}
		rec, err := a.Portal.LatestSmsCode(c.Request.Context(), phone, scene)
		if err != nil {
			respondErr(c, err)
			return
		}
		if rec == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"phone":    rec.Phone,
			"scene":    rec.Scene,
			"code":     rec.Code,
			"issuedAt": rec.IssuedAt,
		})
	}
}