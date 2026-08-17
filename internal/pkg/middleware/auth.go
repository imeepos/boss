package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

const CtxClaims = "boss.claims"

// Authn JWT 认证:解析 Bearer token 并注入 claims。
func Authn(m *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing bearer token"})
			return
		}
		claims, err := m.Verify(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid token"})
			return
		}
		c.Set(CtxClaims, claims)
		c.Next()
	}
}

// PermChecker 权限快照查询(实现走 Redis/DB,权限变更即时生效)。
type PermChecker func(accountID int64, permCode string) (bool, error)

// Authz RBAC 授权:越权访问被拒绝并提示(阶段1验收项)。
func Authz(check PermChecker, permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(CtxClaims).(*auth.Claims)
		ok, err := check(claims.AccountID, permCode)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "permission check failed"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "no permission: " + permCode})
			return
		}
		c.Next()
	}
}
