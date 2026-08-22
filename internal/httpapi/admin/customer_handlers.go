package adminapi

// 客户与资费域路由具名 handler(承接 registerCustomerRoutes 扁平路由表)。
// 客户列表/详情/实名核验日志;产品列表/创建/调价历史。
// 请求体类型 changeProductPriceReq 见 customer.go。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// customerListHandler GET /customers:客户列表(按 keyword/phone/status 过滤,分页)。
func customerListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// customerGetHandler GET /customers/{id}:客户详情;未命中回 NotFound。
func customerGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		det, err := a.Customer.Get(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, det)
	}
}

// customerVerifyLogsHandler GET /customers/{id}/verify-logs:客户实名核验日志。
func customerVerifyLogsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// productListHandler GET /products:产品列表(按 legalEntityId 过滤)。
func productListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Product.ListProducts(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// productCreateHandler POST /products:创建产品;校验必填,防孤儿产品。
func productCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// productPriceHistoryHandler GET /products/{id}/price-history:产品调价历史。
func productPriceHistoryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// productChangePriceHandler POST /products/{id}/price-history:产品调价;更新月费并追加台账。
func productChangePriceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}