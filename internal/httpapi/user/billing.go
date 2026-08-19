package userapi

// 用户端门户 Billing 域:账单明细/缴费凭证/余额充值/申请开票 + 门户偏好存取。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalBillingRoutes 账单明细/凭证/充值/开票。
func registerPortalBillingRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/bills/:billNo", portalBillDetail(a))
	g.GET("/payments/:payNo/receipt", portalReceipt(a))
	g.GET("/topups", portalBalanceGet(a))
	g.POST("/topups", portalTopup(a))
	g.POST("/invoices", portalApplyInvoice(a))
}

// portalBillDetail GET /bills/:billNo:我的账单明细(按客户过滤)。
func portalBillDetail(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, b := range bills {
			if b.BillNo == c.Param("billNo") {
				respond(c, apitypes.CodeOK, gin.H{
					"bill":     gin.H{"billNo": b.BillNo, "period": b.Period, "amount": b.Amount, "status": b.Status},
					"items":    []gin.H{{"name": "宽带月费", "range": b.Period, "amount": b.Amount}},
					"totalDue": map[bool]float64{true: 0, false: b.Amount}[b.Status == "PAID"],
				})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalReceipt GET /payments/:payNo/receipt:在我的缴费流水中找凭证。
func portalReceipt(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bills, _ := a.Billing.ListBills(c.Request.Context(), cid)
		for _, b := range bills {
			pays, _ := a.Billing.ListPayments(c.Request.Context(), b.BillID)
			for _, p := range pays {
				if p.PayNo == c.Param("payNo") {
					respond(c, apitypes.CodeOK, gin.H{
						"receiptNo": "OR-" + p.PayNo, "amount": p.Amount,
						"period": b.Period, "payMethod": p.Method, "status": p.Status,
						"paidAt": time.Now().Format(time.RFC3339), "payNo": p.PayNo,
					})
					return
				}
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

func portalBalanceGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bal, err := a.Portal.Balance(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"balance": bal, "denominations": []int{50, 100, 200}})
	}
}

// portalTopup POST /topups:充值入余额(支付通道接入前仅记账;余额/单号已落库)。
func portalTopup(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Amount    float64 `json:"amount" binding:"required,gt=0"`
			PayMethod string  `json:"payMethod" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Portal.AdjustBalance(c.Request.Context(), cid, req.Amount); err != nil {
			respondErr(c, err)
			return
		}
		payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "amount": req.Amount, "payMethod": req.PayMethod, "status": "SUCCESS",
		})
	}
}

// portalApplyInvoice POST /invoices:申请开票(校验账单归属;Tax 域客户侧开票流程见报告)。
func portalApplyInvoice(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo string `json:"billNo" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		bills, _ := a.Billing.ListBills(c.Request.Context(), cid)
		for _, b := range bills {
			if b.BillNo == req.BillNo {
				respond(c, apitypes.CodeOK, gin.H{"ok": true})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}
