package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// Register 注册管理端路由:/api/v1 前缀 + 账密 JWT/API key 鉴权链。
// admin 端为封闭账号模型:无自助注册,账号由超管引导(EnsureSuperAdmin)或 org/account 受权流程创建。
func Register(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	api := r.Group("/api/admin/v1")
	api.POST("/auth/login", adminLoginHandler(a, mgr))

	// 需要鉴权的路由组:先尝试 API key 免登录认证,再回退 JWT 认证。
	// API key 与三类主体(account/worker/customer)绑定;account 注入完整 RBAC 身份,
	// worker/customer 注入受限身份(菜单门禁 403,扫码接口经 Subject 识别)。
	authed := api.Group("")
	authed.Use(middleware.APIKeyAuth(a.APIKey, httpx.APIKeySubjectResolver(a)), middleware.Authn(mgr, auth.AudAdmin))

	authed.GET("/auth/me", adminMeHandler(a))
	// 退出登录:token 无状态,前端清本地 token 即可(auth.yaml adminLogout)。
	authed.POST("/auth/logout", adminLogoutHandler())
	// 自助改密:校验旧口令后更新;API key 主体无账号概念,拒绝。
	authed.POST("/auth/change-password", adminChangePasswordHandler(a))
	// 自助改基本资料:仅 realName/phone;组织/角色仍走受权流程(menu:account)。
	authed.PUT("/auth/profile", adminUpdateSelfProfileHandler(a))
	// 滑动续期:token 仍有效时换发新 token(TTL 重置),实现"一次登录、活跃期免二次登录"。
	// 角色取 DB 最新快照(权限/角色变更即时生效);API key 主体无账号概念,不参与续期。
	authed.POST("/auth/refresh", adminRefreshTokenHandler(a, mgr))

	registerOrgRoutes(authed, a)
	registerAddressRoutes(authed, a)
	registerSysRoutes(authed, a)
	registerAuthConfigRoutes(authed, a)
	registerSMSConfigRoutes(authed, a)
	registerPushConfigRoutes(authed, a)
	registerRealIDConfigRoutes(authed, a)
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
	registerODNRoutes(authed, a)
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
	registerAttachmentRoutes(authed, a)
	registerNotifyRoutes(authed, a)
	registerBackupRoutes(authed, a)
}