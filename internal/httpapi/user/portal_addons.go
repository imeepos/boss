package userapi

// 用户端门户增值服务域:订购/退订(userdata 订购关系落库)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalAddonSubscribe 订购增值服务(userdata 订购关系落库)。
func portalAddonSubscribe(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if err := portalAddonOpPersist(c, a, cid, "subscribe"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalAddonUnsubscribe 退订增值服务(userdata 退订关系落库)。
func portalAddonUnsubscribe(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if err := portalAddonOpPersist(c, a, cid, "unsubscribe"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalAddonOpPersist 增值订购/退订落库(addon_subscriptions 快照关系)。
func portalAddonOpPersist(c *gin.Context, a *app.Application, cid int64, action string) error {
	if a.UserData == nil {
		return nil
	}
	_, err := a.UserData.CreateAddonSubscription(c.Request.Context(), udcustomer.AddonSubscription{
		CustomerID: cid, AddonID: c.Param("addonId"), Action: action,
	})
	return err
}
