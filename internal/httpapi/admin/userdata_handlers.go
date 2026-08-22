package adminapi

// 用户端数据域路由(列表族)具名 handler:用户主档/账户/地址/套餐/增值服务。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func udListUsers(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUsers(c.Request.Context(), c.Query("keyword"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udGetUserDetail(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
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
	}
}

func udUpdateUserAccount(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
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
	}
}

func udListUserAccounts(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserAccounts(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udListUserAddresses(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserAddresses(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateUserAddress(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
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
	}
}

func udListUserPlans(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserPlans(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateUserPlan(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
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
	}
}

func udListAddons(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListAddons(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := userdata.AssertListContract("ListAddons", "addonId", list); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateAddon(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var ad userdata.Addon
		if !httpx.BindBody(c, &ad) {
			return
		}
		if err := ud.CreateAddon(c.Request.Context(), ad); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udToggleAddon(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		if err := ud.ToggleAddon(c.Request.Context(), c.Param("addonId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListAddonSubscriptions(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListAddonSubscriptions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateAddonSubscription(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
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
	}
}
