package adminapi

// 客户与资费域路由注册(承接 api/openapi/admin/customer.yaml)。
// 全部 handler 实现见 customer_handlers.go。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerCustomerRoutes 注册客户与资费域路由(承接 api/openapi/admin/customer.yaml)。
func registerCustomerRoutes(g *gin.RouterGroup, a *app.Application) {
	cus := g.Group("/customers", requirePerm(a.User, "menu:customer"))
	cus.GET("", customerListHandler(a))
	cus.POST("", customerCreateHandler(a))
	// 代客开户内联建址(2026-09-01):与 /orders/address 同 handler 同契约,仅门禁与
	// customerId 必填性不同(此处可省=未建档);先建址后开户,装机地址弹框内闭环。
	cus.POST("/address", inlineAddressHandler(a, false))
	cus.GET("/:id", customerGetHandler(a))
	cus.GET("/:id/verify-logs", customerVerifyLogsHandler(a))

	// 产品资费读写分离(迁移 000169,何平 2026-08-28 裁定):menu:product=读(目录可见),
	// menu:product-write=写(建档/编辑/状态/调价,仅 sysadmin);ops 保留读保住受理体验。
	prod := g.Group("/products", requirePerm(a.User, "menu:product"))
	prod.GET("", productListHandler(a))
	prod.POST("", requirePerm(a.User, "menu:product-write"), productCreateHandler(a))
	prod.PUT("/:id", requirePerm(a.User, "menu:product-write"), productUpdateHandler(a))
	prod.PUT("/:id/status", requirePerm(a.User, "menu:product-write"), productUpdateStatusHandler(a))
	prod.GET("/:id/price-history", productPriceHistoryHandler(a))
	prod.POST("/:id/price-history", requirePerm(a.User, "menu:product-write"), productChangePriceHandler(a))
	// 产品↔下发模板绑定(方案B):读 menu:product,写 menu:product-write。
	prod.GET("/:id/provision-binding", productGetProvisionBindingHandler(a))
	prod.PUT("/:id/provision-binding", requirePerm(a.User, "menu:product-write"), productUpsertProvisionBindingHandler(a))
	prod.DELETE("/:id/provision-binding", requirePerm(a.User, "menu:product-write"), productDeleteProvisionBindingHandler(a))
}

// changeProductPriceReq 产品调价请求体(对齐 customer.yaml changeProductPrice)。
type changeProductPriceReq struct {
	NewPrice    float64   `json:"newPrice"`
	EffectiveAt time.Time `json:"effectiveAt"`
	Reason      string    `json:"reason"`
}

// updateProductReq 产品编辑请求体:仅基础信息;月费走调价、状态走 status 口(见 customer.yaml updateProduct)。
type updateProductReq struct {
	Name      string `json:"name"`
	Bandwidth string `json:"bandwidth"`
	Category  string `json:"category"`
}

// updateProductStatusReq 产品上下架请求体(DRAFT/PUBLISHED/OFFLINE)。
type updateProductStatusReq struct {
	Status string `json:"status"`
}
