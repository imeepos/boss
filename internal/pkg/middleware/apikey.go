// Package middleware provides Gin middleware for API key authentication.
//
// API key 认证用于 CLI/pipeline 自动化场景,无需先调用 /auth/login 获取 JWT。
// 通过 X-API-Key 请求头传递密钥,服务端校验后注入绑定主体的身份。
//
// 三类主体(与迁移 000043/000044 对齐):
//   - account  → 注入完整 Claims(AccountID/RoleCode),RBAC 按账号角色判定
//   - worker   → 注入受限 Claims(AccountID=0, RoleCode=worker)+ Subject,扫码类接口取真实师傅身份
//   - customer → 注入受限 Claims(AccountID=0, RoleCode=customer)+ Subject
//
// 隔离保证:worker/customer 主体的 AccountID=0,HasPermission 恒为 false,
// 菜单门禁接口(requirePerm)对其一律 403;仅扫码/工单等接口经 Subject 识别身份。
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// CtxSubject API key 主体上下文键(worker/customer 身份经此传递)。
const CtxSubject = "boss.subject"

// Subject 请求主体身份(由 API key 中间件注入)。
type Subject struct {
	Type string // account | worker | customer
	Ref  int64  // 主体表主键
	Name string // 展示名
}

// SubjectResolver 由 app 层注入:按主体类型加载展示名/角色码。
// account 返回 (username, roleCode);worker/customer 返回 (name, 固定角色码)。
type SubjectResolver func(ctx context.Context, subjType string, ref int64) (name, roleCode string, err error)

// APIKeyAuth 返回 API key 认证中间件。
// 当 X-API-Key 匹配数据库中的有效密钥时,注入对应主体的 claims 并跳过后续 Authn。
// 未提供密钥时回退 JWT 认证;密钥无效返回 401。
func APIKeyAuth(keys apikey.Service, resolve SubjectResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.Next()
			return
		}
		h := sha256Hash(key)
		subj, err := keys.Lookup(c.Request.Context(), h)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid api key"})
			return
		}
		name, roleCode, err := resolve(c.Request.Context(), subj.Type, subj.Ref)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "api key subject not found"})
			return
		}
		claims := &auth.Claims{Username: name, RoleCode: roleCode}
		if subj.Type == apikey.SubjectAccount {
			claims.AccountID = subj.Ref // 管理账号:RBAC 按 accountID 判定
		}
		c.Set(CtxClaims, claims)
		c.Set(CtxSubject, &Subject{Type: subj.Type, Ref: subj.Ref, Name: name})
		go keys.Touch(context.Background(), h)
		c.Next()
	}
}

// SubjectFrom 从请求上下文取 API key 主体;无主体时返回 nil。
func SubjectFrom(c *gin.Context) *Subject {
	if v, ok := c.Get(CtxSubject); ok {
		if s, ok := v.(*Subject); ok {
			return s
		}
	}
	return nil
}

// sha256Hash 返回十六进制 SHA-256 哈希。
func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
