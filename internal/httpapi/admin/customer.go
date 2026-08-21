package adminapi

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerCustomerRoutes 注册客户与资费域路由(承接 api/openapi/admin/customer.yaml)。
func registerCustomerRoutes(g *gin.RouterGroup, a *app.Application) {
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
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
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

	// POST /products 创建产品:校验必填字段,保证 LegalEntityID 有效,防止孤儿产品。
	prod.POST("", func(c *gin.Context) {
		var req customer.ProductOffer
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 128),
				httpx.RequirePositiveID(req.LegalEntityID, "legalEntityId"),
				httpx.RequirePositiveFloat(req.MonthlyFee, "monthlyFee"),
			)
		}) {
			return
		}
		// Bandwidth 可选;Category 空值回退 broadband(由 DB 层处理)。
		id, err := a.Product.CreateProduct(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	prod.GET("/:id/price-history", func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		list, err := a.CustomerLedger.ListProductPriceHistories(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 产品调价(worker.yaml POST /products/{id}/price-history):更新月费并追加台账。
	prod.POST("/:id/price-history", func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req changeProductPriceReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveFloat(req.NewPrice, "newPrice"),
			)
		}) {
			return
		}
		effectiveAt := req.EffectiveAt
		if effectiveAt.IsZero() {
			effectiveAt = time.Now()
		}
		historyID, err := a.Product.ChangeProductPrice(c.Request.Context(),
			id, req.NewPrice, effectiveAt, req.Reason, httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "product.change_price", "product_offer", strconv.FormatInt(id, 10), gin.H{"historyId": historyID})
		respond(c, apitypes.CodeOK, gin.H{"id": historyID})
	})
}

// changeProductPriceReq 产品调价请求体(对齐 customer.yaml changeProductPrice)。
type changeProductPriceReq struct {
	NewPrice    float64   `json:"newPrice"`
	EffectiveAt time.Time `json:"effectiveAt"`
	Reason      string    `json:"reason"`
}
