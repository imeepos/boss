package adminapi

// 采购-库存域 handler 实现;对应 registerProcurementRoutes 扁平路由表。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/procurement"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// helpers — queryInt64/respond/respondErr 见 asset_handlers.go 同包。

func procurementListSuppliers(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Procurement.ListSuppliers(c.Request.Context(), queryInt64(c, "legalEntityId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func procurementCreateSupplier(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in procurement.Supplier
		if err := c.ShouldBindJSON(&in); err != nil {
			respondErr(c, err)
			return
		}
		id, err := a.Procurement.CreateSupplier(c.Request.Context(), in)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func procurementDisableSupplier(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Procurement.DisableSupplier(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func procurementListOrders(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Procurement.ListOrders(c.Request.Context(),
			queryInt64(c, "legalEntityId"), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func procurementCreateOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in procurement.Order
		if err := c.ShouldBindJSON(&in); err != nil {
			respondErr(c, err)
			return
		}
		in.CreatedBy = currentAccountID(c, a)
		created, err := a.Procurement.CreateOrder(c.Request.Context(), in)
		if err != nil {
			respondErr(c, err)
			return
		}
		// 回传单号(后端生成兜底,前端无需拼)
		po, err := a.Procurement.GetOrder(c.Request.Context(), created)
		if err != nil || po == nil {
			respond(c, apitypes.CodeOK, gin.H{"id": created})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": created, "procurementNo": po.ProcurementNo})
	}
}

func procurementSubmitOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Procurement.SubmitOrder(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func procurementCancelOrder(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.Procurement.CancelOrder(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func procurementCreateReceipt(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			procurement.Receipt
			Items []procurement.ReceiptItem `json:"items"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			respondErr(c, err)
			return
		}
		id, err := a.Procurement.CreateReceipt(c.Request.Context(), in.Receipt, in.Items)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func procurementConfirmReceipt(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var in procurement.ReceiptConfirmInput
		if err := c.ShouldBindJSON(&in); err != nil {
			respondErr(c, err)
			return
		}
		accountID := currentAccountID(c, a)
		if err := a.Procurement.ConfirmReceipt(c.Request.Context(), id, accountID, in); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

func procurementListReceipts(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Procurement.ListReceipts(c.Request.Context(), queryInt64(c, "orderId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func procurementListInventory(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Procurement.ListInventory(c.Request.Context(),
			queryInt64(c, "legalEntityId"), c.Query("materialCode"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// currentAccountID 从 JWT context 拿当前账号 ID(走 middleware.CtxClaims 标准路径)。
func currentAccountID(c *gin.Context, _ *app.Application) int64 {
	claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
	return claims.AccountID
}
