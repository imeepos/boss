package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/auth"
)

const CtxClaims = "boss.claims"

// Authn JWT 认证:解析 Bearer token 并注入 claims;端标识 aud 精确匹配
// (auth.AudAdmin/AudUser),跨端 token 一律 401。师傅端走独立 issuer,不经此中间件。
// 如果 claims 已被前序中间件(如 APIKeyAuth)设置,则跳过 JWT 校验。
func Authn(m *auth.Manager, aud string) gin.HandlerFunc {
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
		if claims.Aud != aud {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "token audience mismatch"})
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

// StatusChecker 账号有效性查询(实现走 accounts 表 status)。
type StatusChecker func(ctx context.Context, accountID int64) (bool, error)

// AccountActive 停用账号拒已签发 token:JWT 验签不查库,停用后旧 token 在有效期内
// 仍全权可用(2026-08-28 权限实测缺陷3);API key 主体无账号概念,随 Authz 先例放行。
func AccountActive(check StatusChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if SubjectFrom(c) != nil {
			c.Next()
			return
		}
		v, exists := c.Get(CtxClaims)
		if !exists {
			c.Next()
			return
		}
		claims, ok := v.(*auth.Claims)
		if !ok {
			c.Next()
			return
		}
		active, err := check(c.Request.Context(), claims.AccountID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "account status check failed"})
			return
		}
		if !active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "account disabled"})
			return
		}
		c.Next()
	}
}

// Authz RBAC 授权:受限 API key 先按模板校验，普通账号走角色权限。
func Authz(check PermChecker, permCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if subject := SubjectFrom(c); subject != nil && subject.TemplateCode != "" {
			if subject.TemplateCode == "partner-orders-read" && permCode == "menu:partner-orders" {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "api key template does not allow permission"})
			return
		}
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
