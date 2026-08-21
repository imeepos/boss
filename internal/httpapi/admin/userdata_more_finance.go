package adminapi

// 用户端数据域路由(续)具名 handler:余额/充值/发票/投诉/核验/卖点/账单明细。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func udListUserBalances(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserBalances(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udAdjustUserBalance(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
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
	}
}

func udListTopupDenominations(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListTopupDenominations(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udUpdateTopupDenomination(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var d userdata.TopupDenomination
		if !httpx.BindBody(c, &d) {
			return
		}
		if err := ud.UpdateTopupDenomination(c.Request.Context(), c.Param("denomId"), d); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListUserInvoices(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserInvoices(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCreateUserInvoiceDisabled(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		// POST /user-invoices 已按裁定 D1 停写:发票权威态在 billing 域 invoices 表
		// (bill_id 强关联 + ARN 连续发号,无法按 billNo 直插);开票走 billing.TaxService 出账自动开票/重开。
		respond(c, apitypes.CodeStateInvalid, gin.H{
			"reason": "user_invoices 已停写(裁定 D1),开票请走 billing 域发票流程",
		})
	}
}

func udListUserComplaints(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserComplaints(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udCloseUserComplaint(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		if err := ud.CloseUserComplaint(c.Request.Context(), c.Param("complaintId")); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListUserVerifyRecords(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserVerifyRecords(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udListProductSpecs(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListProductSpecs(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

func udUpdateProductSpec(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		var ps userdata.ProductSpec
		if !httpx.BindBody(c, &ps) {
			return
		}
		if err := ud.UpdateProductSpec(c.Request.Context(), c.Param("productId"), ps); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func udListUserBillItems(a *app.Application) gin.HandlerFunc {
	ud := a.UserData
	return func(c *gin.Context) {
		list, err := ud.ListUserBillItems(c.Request.Context(), c.Query("billNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
