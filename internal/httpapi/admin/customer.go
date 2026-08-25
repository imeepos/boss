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
	cus.GET("/:id", customerGetHandler(a))
	cus.GET("/:id/verify-logs", customerVerifyLogsHandler(a))

	prod := g.Group("/products", requirePerm(a.User, "menu:product"))
	prod.GET("", productListHandler(a))
	prod.POST("", productCreateHandler(a))
	prod.GET("/:id/price-history", productPriceHistoryHandler(a))
	prod.POST("/:id/price-history", productChangePriceHandler(a))
}

// changeProductPriceReq 产品调价请求体(对齐 customer.yaml changeProductPrice)。
type changeProductPriceReq struct {
	NewPrice    float64   `json:"newPrice"`
	EffectiveAt time.Time `json:"effectiveAt"`
	Reason      string    `json:"reason"`
}
