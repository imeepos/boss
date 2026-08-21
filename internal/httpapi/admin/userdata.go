package adminapi

// 用户端数据域路由(承接 api/openapi/admin/userdata.yaml):用户列表/详情聚合 + 实体表管理视图。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// bindBody 绑定 JSON 请求体;失败统一回 InvalidParam。
// pathInt64 解析路径整型参数;失败统一回 InvalidParam(bool=false 表示已响应)。
func pathInt64(c *gin.Context, name string) (int64, bool) {
	return httpx.ParsePathParamInt64(c, name)
}

// registerUserdataRoutes 注册用户端数据域路由:列表族。
func registerUserdataRoutes(g *gin.RouterGroup, a *app.Application) {
	ud := a.UserData

	// 用户主档:列表 + 详情聚合 + 账户设置。
	g.GET("/users", requirePerm(a.User, "menu:user"), func(c *gin.Context) {
		list, err := ud.ListUsers(c.Request.Context(), c.Query("keyword"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.GET("/users/:customerId", requirePerm(a.User, "menu:user"), func(c *gin.Context) {
		id, ok := pathInt64(c, "customerId")
		if !ok {
			return
		}
		detail, err := ud.GetUserDetail(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, detail)
	})
	g.PUT("/users/:customerId/account", requirePerm(a.User, "menu:user"), func(c *gin.Context) {
		id, ok := pathInt64(c, "customerId")
		if !ok {
			return
		}
		var u userdata.UserAccountUpdate
		if !httpx.BindBody(c, &u) {
			return
		}
		if err := ud.UpdateUserAccount(c.Request.Context(), id, u); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 账户/地址/套餐。
	g.GET("/user-accounts", requirePerm(a.User, "menu:user"), func(c *gin.Context) {
		list, err := ud.ListUserAccounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.GET("/user-addresses", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		list, err := ud.ListUserAddresses(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.POST("/user-addresses", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var addr userdata.UserAddress
		if !httpx.BindBody(c, &addr) {
			return
		}
		id, err := ud.CreateUserAddress(c.Request.Context(), addr)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})
	g.GET("/user-plans", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		list, err := ud.ListUserPlans(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.POST("/user-plans", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var p userdata.UserPlan
		if !httpx.BindBody(c, &p) {
			return
		}
		id, err := ud.CreateUserPlan(c.Request.Context(), p)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 增值服务:目录 + 上下架 + 订购记录。
	g.GET("/addons", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		list, err := ud.ListAddons(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.POST("/addons", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var ad userdata.Addon
		if !httpx.BindBody(c, &ad) {
			return
		}
		if err := ud.CreateAddon(c.Request.Context(), ad); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	g.PUT("/addons/:addonId/toggle", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		if err := ud.ToggleAddon(c.Request.Context(), c.Param("addonId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	g.GET("/addon-subscriptions", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		list, err := ud.ListAddonSubscriptions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
	g.POST("/addon-subscriptions", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var sub userdata.AddonSubscription
		if !httpx.BindBody(c, &sub) {
			return
		}
		id, err := ud.CreateAddonSubscription(c.Request.Context(), sub)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})
}
