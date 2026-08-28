package adminapi

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// Register 注册管理端路由:/api/v1 前缀 + 账密 JWT/API key 鉴权链。
// admin 端为封闭账号模型:无自助注册,账号由超管引导(EnsureSuperAdmin)或 org/account 受权流程创建。
func Register(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	authed := registerAdminAuthRoot(r, a, mgr)
	registerAdminAuthRoutes(authed, a, mgr)
	// 业务域路由额外套系统级授权门禁(authed 已含登录态;biz 再校验离线证书)。
	biz := adminBizGroup(authed, a)
	registerAdminDomainRoutes(biz, a)
	// 授权页自身路由(豁免门禁,见 adminBizGroup 豁免清单)。
	registerLicenseRoutes(authed, a)
}

// registerAdminAuthRoot 装配 /api/admin/v1 根组 + 鉴权链。
// 公共:登录端点(无认证);鉴权:先 API key 免登录,再回退 JWT 认证。
// API key 与三类主体(account/worker/customer)绑定;account 注入完整 RBAC 身份,
// worker/customer 注入受限身份(菜单门禁 403,扫码接口经 Subject 识别)。
func registerAdminAuthRoot(r *gin.Engine, a *app.Application, mgr *auth.Manager) *gin.RouterGroup {
	api := r.Group("/api/admin/v1")
	api.POST("/auth/login", adminLoginHandler(a, mgr))
	// 入驻申请公开提交(招商引资,免登录;admin 封闭账号模型的唯一自助入口)。
	registerPartnerPublicRoutes(api, a)
	// 官网内容匿名只读(仅 PUBLISHED,免登录;公开面最小投影)。
	registerSitePublicRoutes(api, a)
	// 客户端最新版匿名读(官网首页下载入口,仅 PUBLISHED)。
	registerClientReleasePublicRoutes(api, a)
	authed := api.Group("")
	authed.Use(
		middleware.APIKeyAuth(a.APIKey, httpx.APIKeySubjectResolver(a)),
		middleware.Authn(mgr, auth.AudAdmin),
		// 停用账号逐请求拒止:JWT 验签不查库,否则停用后旧 token 在有效期内仍全权可用。
		middleware.AccountActive(func(ctx context.Context, accountID int64) (bool, error) {
			return a.User.AccountActive(ctx, accountID)
		}),
	)
	return authed
}

// adminBizGroup 业务子组:套系统级授权门禁。
// 豁免:认证自服务(auth/* 已在 authed 注册,不经过本组)与授权页(license/*)。
// License 为 nil 时门禁完全放行(未启用)。详见 adopted license-gate note。
func adminBizGroup(authed *gin.RouterGroup, a *app.Application) *gin.RouterGroup {
	biz := authed.Group("")
	biz.Use(middleware.LicenseGate(a.License, "/api/admin/v1/license/"))
	return biz
}

// registerAdminAuthRoutes 鉴权组内的自身认证端点(me/logout/改密/改资料/续期)。
func registerAdminAuthRoutes(authed *gin.RouterGroup, a *app.Application, mgr *auth.Manager) {
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
}

// registerAdminDomainRoutes 注册鉴权组内的全部业务域路由。
func registerAdminDomainRoutes(authed *gin.RouterGroup, a *app.Application) {
	registerOrgRoutes(authed, a)
	registerRegionRoutes(authed, a)
	registerAddressRoutes(authed, a)
	registerSysRoutes(authed, a)
	registerAuthConfigRoutes(authed, a)
	registerSMSConfigRoutes(authed, a)
	registerPushConfigRoutes(authed, a)
	registerRealIDConfigRoutes(authed, a)
	registerRealnameReviewRoutes(authed, a)
	registerStripeConfigRoutes(authed, a)
	registerAIRoutes(authed, a)
	registerAPIKeyRoutes(authed, a)
	registerOpenPlatRoutes(authed, a)
	registerOrderRoutes(authed, a)
	registerOrderSubRoutes(authed, a)
	registerCallbackRoutes(authed, a)
	registerCSClosureRoutes(authed, a)
	registerOrderWorkflowRoutes(authed, a)
	registerDispatchRoutes(authed, a)
	registerDashboardRoutes(authed, a)
	registerBillingRoutes(authed, a)
	registerTaxRoutes(authed, a)
	registerCustomerRoutes(authed, a)
	registerCustomerOnboardingRoutes(authed, a)
	registerPartnerRoutes(authed, a)
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
	registerCompTaskRoutes(authed, a)
	registerMetricRoutes(authed, a)
	registerETLRoutes(authed, a)
	registerProvisionRoutes(authed, a)
	registerProvisionSeedRoutes(authed, a)
	registerQuadlinkRoutes(authed, a)
	registerWorkerRoutes(authed, a)
	registerWorkerOnboardingRoutes(authed, a)
	registerProcurementRoutes(authed, a)
	registerInstallLogRoutes(authed, a)
	registerUserdataRoutes(authed, a)
	registerUserdataMoreRoutes(authed, a)
	registerUserdataGapRoutes(authed, a)
	registerPromotionRoutes(authed, a)
	registerLoyRoutes(authed, a)
	registerAttachmentRoutes(authed, a)
	registerNotifyRoutes(authed, a)
	registerBackupRoutes(authed, a)
	registerKnowledgeRoutes(authed, a)
	registerSiteRoutes(authed, a)
	registerClientReleaseRoutes(authed, a)
	registerCrashLogRoutes(authed, a)
	registerSearchRoutes(authed, a)
	registerApiDocsRoutes(authed, a)
}
