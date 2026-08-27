package adminapi

// 客户与资费域路由具名 handler(承接 registerCustomerRoutes 扁平路由表)。
// 客户列表/详情/实名核验日志;产品列表/创建/调价历史。
// 请求体类型 changeProductPriceReq 见 customer.go。

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// customerCreateReq 客户直建请求体(批量导入用;regionName 由 regionId 服务端快照)。
type customerCreateReq struct {
	Name          string `json:"name" binding:"required"`
	Phone         string `json:"phone" binding:"required"`
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	AddressID     int64  `json:"addressId" binding:"required"`
	RegionID      int64  `json:"regionId" binding:"required"`
	IdType        string `json:"idType"`
	IdNo          string `json:"idNo"`
}

// customerCreateHandler POST /customers:客户直建(menu:customer;镜像注册审核通过的主档插行,
// REAL_NAME PENDING / SERVICE ACTIVE 默认,idType 缺省身份证)。
func customerCreateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req customerCreateReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		region, err := a.User.GetRegion(c.Request.Context(), req.RegionID)
		if err != nil {
			respondErr(c, err)
			return
		}
		idType := req.IdType
		if idType == "" {
			idType = "身份证"
		}
		id, err := a.Customer.Create(c.Request.Context(), customer.Customer{
			Name: req.Name, Phone: req.Phone, IdType: idType, IdNo: req.IdNo,
			RealNameStatus: customer.RealNamePending, ServiceStatus: "ACTIVE",
			AddressID: req.AddressID, LegalEntityID: req.LegalEntityID,
			RegionID: region.ID, RegionName: region.Name,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "customer", fmt.Sprint(id), gin.H{"op": "create"})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// customerListHandler GET /customers:客户列表(按 keyword/phone/status 过滤,分页)。
func customerListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, err := a.User.GetDataScope(c.Request.Context(), httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		list, err := a.Customer.List(c.Request.Context(), customer.CustomerQuery{
			NameKeyword:   c.Query("keyword"),
			Phone:         c.Query("phone"),
			Status:        c.Query("status"),
			LegalEntityID: scope.LegalEntityID,
			RegionScope:   scope.RegionScope,
			Limit:         int(queryInt64(c, "limit")),
			Offset:        int(queryInt64(c, "offset")),
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
		// EffectiveAt 未传(如批量导入模板)兜底当前时间,避免落 0001-01-01 零值(与调价 handler 同口径)。
		if req.EffectiveAt.IsZero() {
			req.EffectiveAt = time.Now()
		}
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

// productUpdateHandler PUT /products/{id}:编辑产品基础信息(名称/带宽/分类)。
// 月费不可在此改(必须走调价台账),状态走 /status,公司归属不可改(防区域包孤儿)。
func productUpdateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req updateProductReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Name, "name", 128),
				httpx.RequireEnumOrDefault(&req.Category, "category", "broadband", "broadband", "fusion", "addon"),
			)
		}) {
			return
		}
		if err := a.Product.UpdateProduct(c.Request.Context(), id, req.Name, req.Bandwidth, req.Category); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "product.update", "product_offer", strconv.FormatInt(id, 10), gin.H{"name": req.Name})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// productUpdateStatusHandler PUT /products/{id}/status:上下架;发布即生效刷新 effective_at。
func productUpdateStatusHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var req updateProductStatusReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireEnum(req.Status, "status", "DRAFT", "PUBLISHED", "OFFLINE"),
			)
		}) {
			return
		}
		if err := a.Product.UpdateProductStatus(c.Request.Context(), id, req.Status); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "product.update_status", "product_offer", strconv.FormatInt(id, 10), gin.H{"status": req.Status})
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": req.Status})
	}
}
