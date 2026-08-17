package app

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerCustomerRoutes 注册客户与资费域路由(承接 api/openapi/admin/customer.yaml)。
func registerCustomerRoutes(g *gin.RouterGroup, a *Application) {
	cus := g.Group("/customers", requirePerm(a.User, "menu:customer"))
	cus.GET("", func(c *gin.Context) {
		list, err := a.Customer.List(c.Request.Context(), customer.CustomerQuery{
			NameKeyword: c.Query("keyword"),
			Phone:       c.Query("phone"),
			Status:      c.Query("status"),
			Limit:       int(queryInt64(c, "limit")),
			Offset:      int(queryInt64(c, "offset")),
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	cus.GET("/:id/verify-logs", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		list, err := a.RealName.ListVerifications(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	prod := g.Group("/products", requirePerm(a.User, "menu:product"))
	prod.GET("", func(c *gin.Context) {
		list, err := a.Product.ListProducts(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	prod.POST("", func(c *gin.Context) {
		var req customer.ProductOffer
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id, err := a.Product.CreateProduct(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	prod.GET("/:id/price-history", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		list, err := a.CustomerLedger.ListProductPriceHistories(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
