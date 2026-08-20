package adminapi

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

// Register 注册管理端路由:/api/v1 前缀 + 账密 JWT/API key 鉴权链。
// admin 端为封闭账号模型:无自助注册,账号由超管引导(EnsureSuperAdmin)或 org/account 受权流程创建。
// admin 端为封闭账号模型:无自助注册,账号由超管引导(EnsureSuperAdmin)或 org/account 受权流程创建。
func Register(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	api := r.Group("/api/admin/v1")

	api.POST("/auth/login", func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
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
	})

	// 需要鉴权的路由组:先尝试 API key 免登录认证,再回退 JWT 认证。
	// API key 与三类主体(account/worker/customer)绑定;account 注入完整 RBAC 身份,
	// worker/customer 注入受限身份(菜单门禁 403,扫码接口经 Subject 识别)。
	authed := api.Group("")
	authed.Use(middleware.APIKeyAuth(a.APIKey, httpx.APIKeySubjectResolver(a)), middleware.Authn(mgr, auth.AudAdmin))

	authed.GET("/auth/me", func(c *gin.Context) {
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
		respond(c, apitypes.CodeOK, p)
	})

	// 退出登录:token 无状态,前端清本地 token 即可(auth.yaml adminLogout)。
	authed.POST("/auth/logout", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 自助改密:校验旧口令后更新;API key 主体无账号概念,拒绝。
	authed.POST("/auth/change-password", func(c *gin.Context) {
		if middleware.SubjectFrom(c) != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		var req changePasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.User.ChangePassword(c.Request.Context(), claims.AccountID, req.OldPassword, req.NewPassword); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "权限变更", "account", fmt.Sprint(claims.AccountID), map[string]any{"op": "self-change-password"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 自助改基本资料:仅 realName/phone;组织/角色仍走受权流程(menu:account)。
	authed.PUT("/auth/profile", func(c *gin.Context) {
		if middleware.SubjectFrom(c) != nil {
			respond(c, apitypes.CodeUnauthorized, nil)
			return
		}
		var req selfProfileReq
		if err := c.ShouldBindJSON(&req); err != nil || req.RealName == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.User.UpdateSelfProfile(c.Request.Context(), claims.AccountID, req.RealName, req.Phone); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "account", fmt.Sprint(claims.AccountID), map[string]any{"op": "self-update-profile"})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 滑动续期:token 仍有效时换发新 token(TTL 重置),实现"一次登录、活跃期免二次登录"。
	// 角色取 DB 最新快照(权限/角色变更即时生效);API key 主体无账号概念,不参与续期。
	authed.POST("/auth/refresh", func(c *gin.Context) {
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
	})

	registerOrgRoutes(authed, a)
	registerAddressRoutes(authed, a)
	registerSysRoutes(authed, a)
	registerAuthConfigRoutes(authed, a)
	registerAIRoutes(authed, a)
	registerAPIKeyRoutes(authed, a)
	registerOrderRoutes(authed, a)
	registerOrderSubRoutes(authed, a)
	registerOrderWorkflowRoutes(authed, a)
	registerDispatchRoutes(authed, a)
	registerDashboardRoutes(authed, a)
	registerBillingRoutes(authed, a)
	registerTaxRoutes(authed, a)
	registerCustomerRoutes(authed, a)
	registerCustomerOnboardingRoutes(authed, a)
	registerResourceRoutes(authed, a)
	registerScanRoutes(authed, a)
	registerAssetRoutes(authed, a)
	registerAaaRoutes(authed, a)
	registerDeviceRoutes(authed, a)
	registerGeoRoutes(authed, a)
	registerGisRoutes(authed, a)
	registerAnalyticsRoutes(authed, a)
	registerReportRoutes(authed, a)
	registerProvisionRoutes(authed, a)
	registerProvisionSeedRoutes(authed, a)
	registerQuadlinkRoutes(authed, a)
	registerWorkerRoutes(authed, a)
	registerWorkerOnboardingRoutes(authed, a)
	registerUserdataRoutes(authed, a)
	registerUserdataMoreRoutes(authed, a)
	registerUserdataGapRoutes(authed, a)

}
