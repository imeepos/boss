package userapi

// 用户端门户 Order 域:路由注册 + 产品详情/订单操作/评价的视图工具。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
)

// registerPortalOrderRoutes 客户视角产品/套餐/订单路由入口。
// 各 handler 拆分于 portal_orders.go / portal_plans.go / portal_addons.go / trade_handlers.go。
func registerPortalOrderRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/products", portalListProducts(a))
	g.GET("/products/:id", portalProductDetail(a))
	g.GET("/addons", portalListAddons(a))
	g.POST("/addons/:addonId/subscribe", portalAddonSubscribe(a))
	g.POST("/addons/:addonId/unsubscribe", portalAddonUnsubscribe(a))
	g.GET("/orders", portalListOrders(a))
	g.POST("/orders", portalSubmitOrder(a))
	g.GET("/orders/:orderNo", portalGetOrder(a))
	g.GET("/orders/:orderNo/technician-contact", portalOrderTechnicianContact(a))
	g.POST("/orders/:orderNo/cancel", portalOrderCancel(a))
	g.POST("/orders/:orderNo/urge", portalOrderUrge(a))
	g.POST("/orders/:orderNo/change-address", portalOrderChangeAddr(a))
	g.GET("/orders/:orderNo/rate", portalRateGet(a))
	g.POST("/orders/:orderNo/rate", portalRatePost(a))
	g.GET("/plans/:planId/cancel", portalPlanCancelPreview(a))
	g.POST("/plans/:planId/change", portalPlanChange(a))
	g.POST("/plans/:planId/move", portalPlanMove(a))
	g.POST("/plans/:planId/renew", portalPlanRenew(a))
	g.POST("/plans/:planId/cancel", portalPlanCancel(a))
}

// portalProductSummary 产品主信息视图(列表与详情共用),契约 Product 字段集。
func portalProductSummary(p customer.ProductOffer) gin.H {
	return gin.H{
		"productId": strconv.FormatInt(p.ID, 10), "category": p.Category,
		"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
		"description":    productDescription(p),
		"contractMonths": productContractMonths(p.Category),
		"featured":       productFeatured(p),
	}
}

// productDescription 描述文案:带宽 + 适用人群;DB 无 description 字段时由带宽与分类派生。
func productDescription(p customer.ProductOffer) string {
	bw := p.Bandwidth
	switch p.Category {
	case "broadband":
		return bw + " 高速宽带,适合家庭多设备上网、高清影音"
	case "fusion":
		return bw + " 5G 融合套餐,宽带 + 语音 + 流量一体化"
	default:
		if bw == "" {
			return p.Name
		}
		return bw + " 增值服务"
	}
}

// productContractMonths 合约月数:DB 未建模,按 category 规则约定。
func productContractMonths(category string) int {
	switch category {
	case "broadband":
		return 12
	case "fusion":
		return 24
	default:
		return 1
	}
}

// productFeatured 推荐标记:月费 ≤100 或带宽 ≤500M 视为普通推荐档;高带宽/高价不强制置顶。
func productFeatured(p customer.ProductOffer) bool {
	return p.MonthlyFee > 0 && p.MonthlyFee <= 100
}

// productSpecs 套餐规格行(详情卡片):带宽/合约期/安装费/设备/适用范围。
// DB 未建模的字段(安装费/设备/适用范围)按 category 给出文案占位,前端不改契约即可渲染。
func productSpecs(p customer.ProductOffer) []gin.H {
	bwValue := p.Bandwidth
	if bwValue != "" {
		bwValue += "bps"
	}
	installFee := "¥0"
	device := "光猫免费租用"
	scope := "家庭宽带"
	switch p.Category {
	case "fusion":
		device = "5G CPE 路由器"
		scope = "5G 融合套餐"
	case "addon":
		installFee = "—"
		device = "—"
		scope = "增值服务"
	}
	return []gin.H{
		{"label": "带宽", "value": orDash(bwValue)},
		{"label": "合约期", "value": strconv.Itoa(productContractMonths(p.Category)) + " 个月"},
		{"label": "安装费", "value": installFee},
		{"label": "设备", "value": device},
		{"label": "适用范围", "value": scope},
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
