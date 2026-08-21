package adminapi

// 用户端数据域路由(承接 api/openapi/admin/userdata.yaml):用户列表/详情聚合 + 实体表管理视图。
// 具名 handler 见 userdata_handlers.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// pathInt64 解析路径整型参数;失败统一回 InvalidParam(bool=false 表示已响应)。
func pathInt64(c *gin.Context, name string) (int64, bool) {
	return httpx.ParsePathParamInt64(c, name)
}

// registerUserdataRoutes 注册用户端数据域路由:列表族。
func registerUserdataRoutes(g *gin.RouterGroup, a *app.Application) {
	// 用户主档:列表 + 详情聚合 + 账户设置。
	g.GET("/users", requirePerm(a.User, "menu:user"), udListUsers(a))
	g.GET("/users/:customerId", requirePerm(a.User, "menu:user"), udGetUserDetail(a))
	g.PUT("/users/:customerId/account", requirePerm(a.User, "menu:user"), udUpdateUserAccount(a))

	// 账户/地址/套餐。
	g.GET("/user-accounts", requirePerm(a.User, "menu:user"), udListUserAccounts(a))
	g.GET("/user-addresses", requirePerm(a.User, "menu:userdata"), udListUserAddresses(a))
	g.POST("/user-addresses", requirePerm(a.User, "menu:userdata"), udCreateUserAddress(a))
	g.GET("/user-plans", requirePerm(a.User, "menu:userdata"), udListUserPlans(a))
	g.POST("/user-plans", requirePerm(a.User, "menu:userdata"), udCreateUserPlan(a))

	// 增值服务:目录 + 上下架 + 订购记录。
	g.GET("/addons", requirePerm(a.User, "menu:userdata"), udListAddons(a))
	g.POST("/addons", requirePerm(a.User, "menu:userdata"), udCreateAddon(a))
	g.PUT("/addons/:addonId/toggle", requirePerm(a.User, "menu:userdata"), udToggleAddon(a))
	g.GET("/addon-subscriptions", requirePerm(a.User, "menu:userdata"), udListAddonSubscriptions(a))
	g.POST("/addon-subscriptions", requirePerm(a.User, "menu:userdata"), udCreateAddonSubscription(a))
}
