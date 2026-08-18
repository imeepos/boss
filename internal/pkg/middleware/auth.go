package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

const CtxClaims = "boss.claims"

// Authn JWT 认证:解析 Bearer token 并注入 claims。
// 如果 claims 已被前序中间件(如 APIKeyAuth)设置,则跳过 JWT 校验。
func Authn(m *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get(CtxClaims); ok {
			c.Next()
			return
		}
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
type PermChecker func(ctx context.Context, accountID int64, permCode string) (bool, error)

// DataScopeChecker 数据范围判定:资源属主组织是否落在账号数据范围内。
type DataScopeChecker func(ctx context.Context, accountID int64, owner any) (bool, error)

// Authz RBAC 授权:越权访问被拒绝并提示(阶段1验收项)。
func Authz(check PermChecker, permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(CtxClaims).(*auth.Claims)
		ok, err := check(c.Request.Context(), claims.AccountID, permCode)
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

// DataAuthz 数据范围授权:超出账号数据范围的组织访问被拒绝并提示(阶段1扩展验收项)。
func DataAuthz(check DataScopeChecker, owner any) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(CtxClaims).(*auth.Claims)
		ok, err := check(c.Request.Context(), claims.AccountID, owner)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "data scope check failed"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "out of data scope"})
			return
		}
		c.Next()
	}
}
