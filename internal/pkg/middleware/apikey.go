// Package middleware provides Gin middleware for API key authentication.
//
// API key 认证用于 CLI/pipeline 自动化场景,无需先调用 /auth/login 获取 JWT。
// 通过 X-API-Key 请求头传递密钥,服务端校验后注入绑定的账号身份 claims。
// 权限随账号角色走 RBAC,与普通登录用户一致。
package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// APIKeyAuth 返回 API key 认证中间件。
// 当 X-API-Key 匹配数据库中的有效密钥时,注入对应账号的 claims 并跳过后续 Authn。
// 未提供或密钥无效时,回退到 JWT 认证(由后续 Authn 中间件处理)。
func APIKeyAuth(keys apikey.Service, usr user.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.Next()
			return
		}
		h := sha256Hash(key)
		accountID, err := keys.Lookup(c.Request.Context(), h)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid api key"})
			return
		}
		// 加载账号信息构建 claims
		profile, err := usr.GetProfile(c.Request.Context(), accountID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "api key account not found"})
			return
		}
		c.Set(CtxClaims, &auth.Claims{
			AccountID: accountID,
			Username:  profile.Username,
			RoleCode:  profile.RoleCode,
		})
		// 异步更新 last_used_at(尽力而为)
		go keys.Touch(c.Request.Context(), h)
		c.Next()
	}
}

// sha256Hash 返回十六进制 SHA-256 哈希。
func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}