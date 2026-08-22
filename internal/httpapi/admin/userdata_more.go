package adminapi

// 用户端数据域路由(续):通知/FAQ/消息/优惠券/邀请/用量/指南/协议/余额/充值/发票/投诉/核验/卖点/账单明细。
// 具名 handler 拆到 userdata_more_settings.go(配置族)与 userdata_more_finance.go(账务族)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// udList 通用列表端点:menu 权限 + 服务方法 + envelope {items}。
// 仍供 userdata_gap.go 中的 /faqs 等复用,保持行为/接口完全不变。
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
	perm := requirePerm(a.User, "menu:userdata")
	g.GET("/user-notify-settings", perm, udListNotifySettings(a))
	g.PUT("/user-notify-settings/:customerId", perm, udUpdateNotifySettings(a))
	g.GET("/user-faqs", perm, udListUserFaqs(a))
	g.POST("/user-faqs", perm, udCreateUserFaq(a))
	g.PUT("/user-faqs/:faqId/toggle", perm, udToggleUserFaq(a))
	g.GET("/user-messages", perm, udListUserMessages(a))
	g.POST("/user-messages", perm, udCreateUserMessage(a))
	g.PUT("/user-messages/read-all", perm, udMarkAllMessagesRead(a))
	g.GET("/coupons", perm, udListCoupons(a))
	g.POST("/coupons", perm, udCreateCoupon(a))
	g.PUT("/coupons/:couponId/disable", perm, udDisableCoupon(a))
	g.GET("/invite-config", perm, udListInviteConfig(a))
	g.PUT("/invite-config", perm, udUpdateInviteConfig(a))
	g.GET("/user-usages", perm, udListUserUsages(a))
	g.GET("/diy-guides", perm, udListDiyGuides(a))
	g.PUT("/diy-guides/:guideId/toggle", perm, udToggleDiyGuide(a))
	g.GET("/agreements", perm, udListAgreements(a))
	g.PUT("/agreements/:agreementId", perm, udUpdateAgreement(a))
	g.GET("/user-balances", perm, udListUserBalances(a))
	g.POST("/user-balances/:customerId/adjust", perm, udAdjustUserBalance(a))
	g.GET("/topup-denominations", perm, udListTopupDenominations(a))
	g.PUT("/topup-denominations/:denomId", perm, udUpdateTopupDenomination(a))
	g.GET("/user-invoices", perm, udListUserInvoices(a))
	g.POST("/user-invoices", perm, udCreateUserInvoiceDisabled(a))
	g.GET("/user-complaints", perm, udListUserComplaints(a))
	g.POST("/user-complaints/:complaintId/close", perm, udCloseUserComplaint(a))
	g.GET("/user-verify-records", perm, udListUserVerifyRecords(a))
	g.GET("/product-specs", perm, udListProductSpecs(a))
	g.PUT("/product-specs/:productId", perm, udUpdateProductSpec(a))
	g.GET("/user-bill-items", perm, udListUserBillItems(a))
}
