package adminapi

// 用户端数据域路由(续):通知/FAQ/消息/优惠券/邀请/用量/指南/协议/余额/充值/发票/投诉/核验/卖点/账单明细。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// udList 通用列表端点:menu 权限 + 服务方法 + envelope {items}。
func udList(g *gin.RouterGroup, a *app.Application, path, perm string,
	fn func(c *gin.Context) ([]map[string]any, error)) {
	g.GET(path, requirePerm(a.User, perm), func(c *gin.Context) {
		list, err := fn(c)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}

// registerUserdataMoreRoutes 注册用户端数据域路由:配置与记录族。
func registerUserdataMoreRoutes(g *gin.RouterGroup, a *app.Application) {
	ud := a.UserData

	udList(g, a, "/user-notify-settings", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListNotifySettings(c.Request.Context())
	})
	g.PUT("/user-notify-settings/:customerId", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
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
	})

	udList(g, a, "/user-faqs", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserFaqs(c.Request.Context())
	})
	g.POST("/user-faqs", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var f userdata.UserFaq
		if !httpx.BindBody(c, &f) {
			return
		}
		if err := ud.CreateUserFaq(c.Request.Context(), f); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	g.PUT("/user-faqs/:faqId/toggle", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		if err := ud.ToggleUserFaq(c.Request.Context(), c.Param("faqId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/user-messages", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserMessages(c.Request.Context(), c.Query("keyword"))
	})
	g.POST("/user-messages", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
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
	})
	g.PUT("/user-messages/read-all", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
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
	})

	udList(g, a, "/coupons", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListCoupons(c.Request.Context())
	})
	g.POST("/coupons", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var cp userdata.Coupon
		if !httpx.BindBody(c, &cp) {
			return
		}
		if err := ud.CreateCoupon(c.Request.Context(), cp); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	g.PUT("/coupons/:couponId/disable", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		if err := ud.DisableCoupon(c.Request.Context(), c.Param("couponId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/invite-config", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.GetInviteConfig(c.Request.Context())
	})
	udList(g, a, "/user-usages", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserUsages(c.Request.Context())
	})
	udList(g, a, "/diy-guides", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListDiyGuides(c.Request.Context())
	})
	g.PUT("/diy-guides/:guideId/toggle", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		if err := ud.ToggleDiyGuide(c.Request.Context(), c.Param("guideId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/agreements", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListAgreements(c.Request.Context())
	})
	g.PUT("/agreements/:agreementId", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var ag userdata.Agreement
		if !httpx.BindBody(c, &ag) {
			return
		}
		if err := ud.UpdateAgreement(c.Request.Context(), c.Param("agreementId"), ag); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/user-balances", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserBalances(c.Request.Context())
	})
	g.POST("/user-balances/:customerId/adjust", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		id, ok := pathInt64(c, "customerId")
		if !ok {
			return
		}
		var adj userdata.BalanceAdjust
		if !httpx.BindBody(c, &adj) {
			return
		}
		if err := ud.AdjustUserBalance(c.Request.Context(), id, adj.Delta); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/topup-denominations", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListTopupDenominations(c.Request.Context())
	})
	g.PUT("/topup-denominations/:denomId", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var d userdata.TopupDenomination
		if !httpx.BindBody(c, &d) {
			return
		}
		if err := ud.UpdateTopupDenomination(c.Request.Context(), c.Param("denomId"), d); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/user-invoices", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserInvoices(c.Request.Context())
	})
	g.POST("/user-invoices", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var inv userdata.UserInvoice
		if !httpx.BindBody(c, &inv) {
			return
		}
		id, err := ud.CreateUserInvoice(c.Request.Context(), inv)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	udList(g, a, "/user-complaints", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserComplaints(c.Request.Context())
	})
	g.POST("/user-complaints/:complaintId/close", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		if err := ud.CloseUserComplaint(c.Request.Context(), c.Param("complaintId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/user-verify-records", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserVerifyRecords(c.Request.Context())
	})
	udList(g, a, "/product-specs", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListProductSpecs(c.Request.Context())
	})
	g.PUT("/product-specs/:productId", requirePerm(a.User, "menu:userdata"), func(c *gin.Context) {
		var ps userdata.ProductSpec
		if !httpx.BindBody(c, &ps) {
			return
		}
		if err := ud.UpdateProductSpec(c.Request.Context(), c.Param("productId"), ps); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	udList(g, a, "/user-bill-items", "menu:userdata", func(c *gin.Context) ([]map[string]any, error) {
		return ud.ListUserBillItems(c.Request.Context(), c.Query("billNo"))
	})
}
