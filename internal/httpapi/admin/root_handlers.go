package adminapi

// 管理端鉴权域 handler 实现(从 root.go 抽出,Register 只剩扁平路由表)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// loginReq 登录请求体(对齐 api/openapi/admin/auth.yaml)。
type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// changePasswordReq 自助改密请求体:旧口令校验 + 新口令最短 6 位。
type changePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

// selfProfileReq 自助改基本资料请求体:realName 必填,phone 可空(清空)。
type selfProfileReq struct {
	RealName string `json:"realName" binding:"required"`
	Phone    string `json:"phone"`
}

// adminLoginHandler POST /auth/login:账号密码登录。
func adminLoginHandler(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Username, "username", 64),
				httpx.RequireString(req.Password, "password", 128),
			)
		}) {
			return
		}
		res, err := a.User.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			respondErr(c, err)
			return
		}
		token, err := mgr.Sign(auth.AudAdmin, res.AccountID, res.Username, res.RoleCode)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"token":     token,
			"accountId": res.AccountID,
			"realName":  res.RealName,
			"roleName":  res.RoleName,
		})
	}
}

// adminMeHandler GET /auth/me:当前账号/主体信息。
// 受限模板 API key(account 主体+templateCode)回模板授权码集而非角色全集:
// Authz 对模板 key 只放行模板码,profile 回角色全集会令调用方高估可用面(bossmcp 目录过滤依赖此真相)。
func adminMeHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		// API key worker/customer 主体:返回主体身份(非账号,无 RBAC profile)
		if s := middleware.SubjectFrom(c); s != nil && s.Type != apikey.SubjectAccount {
			respond(c, apitypes.CodeOK, gin.H{
				"subjectType": s.Type, "subjectRef": s.Ref, "name": s.Name,
			})
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		p, err := a.User.GetProfile(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if s := middleware.SubjectFrom(c); s != nil && s.TemplateCode != "" {
			codes, terr := templatePermCodes(c, a, s.TemplateCode)
			if terr != nil {
				respondErr(c, terr)
				return
			}
			p.PermissionCodes = codes
			p.TemplateCode = s.TemplateCode
		}
		respond(c, apitypes.CodeOK, p)
	}
}

// templatePermCodes 受限模板的授权码集;未知模板返回空集(与 Authz 恒拒一致)。
func templatePermCodes(c *gin.Context, a *app.Application, code string) ([]string, error) {
	templates, err := a.APIKey.ListTemplates(c.Request.Context())
	if err != nil {
		return nil, err
	}
	for _, t := range templates {
		if t.Code == code {
			return t.Permissions, nil
		}
	}
	return []string{}, nil
}

// adminLogoutHandler POST /auth/logout:无状态 JWT,前端清本地 token 即可。
func adminLogoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// adminChangePasswordHandler POST /auth/change-password:自助改密。
func adminChangePasswordHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.SubjectFrom(c) != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		var req changePasswordReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.OldPassword, "oldPassword", 128),
				httpx.RequireString(req.NewPassword, "newPassword", 128),
			)
		}) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.User.ChangePassword(c.Request.Context(), claims.AccountID, req.OldPassword, req.NewPassword); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", fmt.Sprint(claims.AccountID), map[string]any{"op": "self-change-password"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// adminUpdateSelfProfileHandler PUT /auth/profile:自助改基本资料。
func adminUpdateSelfProfileHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.SubjectFrom(c) != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		var req selfProfileReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.RealName, "realName", 64),
			)
		}) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.User.UpdateSelfProfile(c.Request.Context(), claims.AccountID, req.RealName, req.Phone); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "account", fmt.Sprint(claims.AccountID), map[string]any{"op": "self-update-profile"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// adminRefreshTokenHandler POST /auth/refresh:滑动续期。
func adminRefreshTokenHandler(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.SubjectFrom(c) != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		p, err := a.User.GetProfile(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		token, err := mgr.Sign(auth.AudAdmin, p.AccountID, p.Username, p.RoleCode)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"token": token})
	}
}
