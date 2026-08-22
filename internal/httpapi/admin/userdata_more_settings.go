package adminapi

// 用户端数据域路由(续)具名 handler:通知/FAQ/消息/优惠券/邀请/用量/指南/协议。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func udListNotifySettings(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListNotifySettings(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := userdata.AssertListContract("ListNotifySettings", "customerId", list); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udUpdateNotifySettings(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		id, ok := pathInt64(c, "customerId")
		if !ok {
			return
		}
		var n userdata.NotifySetting
		if !httpx.BindBody(c, &n) {
			return
		}
		if err := ud.UpdateNotifySettings(c.Request.Context(), id, n); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListUserFaqs(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserFaqs(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := userdata.AssertListContract("ListUserFaqs", "faqId", list); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateUserFaq(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var f userdata.UserFaq
		if !httpx.BindBody(c, &f) {
			return
		}
		if err := ud.CreateUserFaq(c.Request.Context(), f); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udToggleUserFaq(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		if err := ud.ToggleUserFaq(c.Request.Context(), c.Param("faqId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListUserMessages(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserMessages(c.Request.Context(), c.Query("keyword"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateUserMessage(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var m userdata.UserMessage
		if !httpx.BindBody(c, &m) {
			return
		}
		id, err := ud.CreateUserMessage(c.Request.Context(), m)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func udMarkAllMessagesRead(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var req struct {
			CustomerID int64 `json:"customerId" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := ud.MarkAllMessagesRead(c.Request.Context(), req.CustomerID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListCoupons(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListCoupons(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := userdata.AssertListContract("ListCoupons", "couponId", list); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateCoupon(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var cp userdata.Coupon
		if !httpx.BindBody(c, &cp) {
			return
		}
		if err := ud.CreateCoupon(c.Request.Context(), cp); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udDisableCoupon(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		if err := ud.DisableCoupon(c.Request.Context(), c.Param("couponId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListInviteConfig(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.GetInviteConfig(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := userdata.AssertListContract("ListInviteConfig", "id", list); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udListUserUsages(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserUsages(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udListDiyGuides(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListDiyGuides(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := userdata.AssertListContract("ListDiyGuides", "guideId", list); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udToggleDiyGuide(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		if err := ud.ToggleDiyGuide(c.Request.Context(), c.Param("guideId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListAgreements(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListAgreements(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udUpdateAgreement(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var ag userdata.Agreement
		if !httpx.BindBody(c, &ag) {
			return
		}
		if err := ud.UpdateAgreement(c.Request.Context(), c.Param("agreementId"), ag); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
