package adminapi

// 客户注册/审核/实名核验 子域路由注册(迁移 000051)。
// 注册申请为公开端点,已在 RegisterRoutes 的 api 组注册;此处为审核队列 + 实名核验(均走 menu:customer)。
// 全部 handler 实现见 customer_onboarding_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerCustomerOnboardingRoutes 注册客户注册 / 审核 / 实名认证 子域路由(迁移 000051)。
func registerCustomerOnboardingRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/customer-registrations", requirePerm(a.User, "menu:customer"), customerRegistrationListHandler(a))
	g.GET("/customer-registrations/source-stats", requirePerm(a.User, "menu:customer"), customerSourceStatsHandler(a))
	g.POST("/customer-registrations/:id/approve", requirePerm(a.User, "menu:customer"), customerRegistrationApproveHandler(a))
	g.POST("/customer-registrations/:id/reject", requirePerm(a.User, "menu:customer"), customerRegistrationRejectHandler(a))

	g.POST("/customers/:id/real-name", requirePerm(a.User, "menu:customer"), customerSubmitRealNameHandler(a))
	g.GET("/customers/:id/real-name", requirePerm(a.User, "menu:customer"), customerGetRealNameHandler(a))
	g.POST("/customers/:id/real-name/verify", requirePerm(a.User, "menu:customer"), customerVerifyRealNameHandler(a))
}

// customerRegistrationReq 客户注册申请请求体。
