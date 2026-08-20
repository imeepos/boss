package userapi

// 用户端门户 Billing 域:账单明细/缴费凭证/余额充值/申请开票 + 门户偏好存取。

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalBillingRoutes 账单/流水/凭证/充值/开票。
func registerPortalBillingRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/bills", portalListBills(a))
	g.GET("/bills/:billNo", portalBillDetail(a))
	g.GET("/payments", portalListPayments(a))
	g.POST("/payments", portalCreatePayment(a))
	g.GET("/payments/:payNo/receipt", portalReceipt(a))
	g.GET("/payments/:payNo/receipt.pdf", portalReceiptPdf(a))
	g.GET("/billing/auto-pay", portalAutoPayGet(a))
	g.POST("/billing/auto-pay", portalAutoPaySet(a))
	g.GET("/topups", portalBalanceGet(a))
	g.POST("/topups", portalTopup(a))
	g.GET("/invoices", portalListInvoices(a))
	g.POST("/invoices", portalApplyInvoice(a))
	g.GET("/invoices/:invoiceNo/pdf", portalInvoicePdf(a))
}

// portalBillStatusLabel 账单状态中文标签。
var portalBillStatusLabel = map[string]string{"UNPAID": "未缴", "PAID": "已缴", "OVERDUE": "逾期"}

// portalBill 账单座 → 契约 Bill。
func portalBill(b billing.Bill, productName string) gin.H {
	return gin.H{
		"billNo": b.BillNo, "period": b.Period, "productName": productName,
		"periodRange": periodRange(b.Period), "amount": b.Amount,
		"status": b.Status, "statusLabel": portalBillStatusLabel[b.Status],
	}
}

// periodRange 账期 2026-08 → 08-01 ~ 08-31。
func periodRange(period string) string {
	if len(period) >= 7 {
		return period[5:7] + "-01 ~ " + period[5:7] + "-31"
	}
	return period
}

// portalListBills GET /bills?status=:我的账单列表(currentDue/currentPeriod/items)。
func portalListBills(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		status := c.DefaultQuery("status", "recent")
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		currentDue, currentPeriod := 0.0, ""
		items := make([]gin.H, 0, len(bills))
		for _, b := range bills {
			if b.Status == "PAID" {
				if status == "unpaid" || status == "recent" {
					continue
				}
			} else if status == "paid" {
				continue
			}
			items = append(items, portalBill(b, planProductName(a, c, cid)))
			if b.Status != "PAID" {
				currentDue += b.Amount
			}
			if b.Period > currentPeriod {
				currentPeriod = b.Period
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"currentDue": currentDue, "currentPeriod": currentPeriod, "items": items,
		})
	}
}

// planProductName 客户当前套餐名(bills 无产品名快照,以 user_plans 回填)。
func planProductName(a *app.Application, c *gin.Context, cid int64) string {
	if a.UserData == nil {
		return "宽带月费"
	}
	rows, err := a.UserData.ListUserPlans(c.Request.Context())
	if err != nil {
		return "宽带月费"
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) == cid {
			if name := toStr(r["planName"]); name != "" {
				return name
			}
		}
	}
	return "宽带月费"
}

// portalCreatePayment POST /payments:发起缴费(bill 归属校验 + RecordPayment 落库置 PAID)。
func portalCreatePayment(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo    string  `json:"billNo" binding:"required"`
			Amount    float64 `json:"amount" binding:"required,gt=0"`
			PayMethod string  `json:"payMethod" binding:"required,oneof=wechat alipay card cash"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, b := range bills {
			if b.BillNo == req.BillNo {
				if b.Status == "PAID" {
					respond(c, apitypes.CodeConflict, nil)
					return
				}
				payNo, err := a.Portal.NextNo(c.Request.Context(), "PAY")
				if err != nil {
					respondErr(c, err)
					return
				}
				if _, err := a.Billing.RecordPayment(c.Request.Context(), billing.Payment{
					PayNo: payNo, BillID: b.BillID, Amount: req.Amount,
					Method: req.PayMethod, Status: "SUCCESS",
				}); err != nil {
					respondErr(c, err)
					return
				}
				respond(c, apitypes.CodeOK, gin.H{
					"payNo": payNo, "amount": req.Amount, "billPeriod": b.Period,
					"payMethod": req.PayMethod, "status": "SUCCESS",
				})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalListPayments GET /payments:我的缴费记录(缴费+充值,按客户聚合)。
func portalListPayments(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		pays, err := a.Billing.ListPaymentsByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		periodByBill := portalBillPeriods(a, c, cid)
		items := make([]gin.H, 0, len(pays))
		for _, p := range pays {
			period := "余额充值"
			if p.BillID != 0 {
				period = periodByBill[p.BillID]
			}
			items = append(items, gin.H{
				"payNo": p.PayNo, "amount": p.Amount, "period": period,
				"payMethod": p.Method, "paidAt": time.Now().Format(time.RFC3339),
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalBillPeriods 客户账单 billID → 账期映射(缴费流水回显账期用)。
func portalBillPeriods(a *app.Application, c *gin.Context, cid int64) map[int64]string {
	out := map[int64]string{}
	bills, err := a.Billing.ListBills(c.Request.Context(), cid)
	if err != nil {
		return out
	}
	for _, b := range bills {
		out[b.BillID] = b.Period
	}
	return out
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

// portalReceipt GET /payments/:payNo/receipt:在我的缴费流水中找凭证(含充值)。
func portalReceipt(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		pays, err := a.Billing.ListPaymentsByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		periodByBill := portalBillPeriods(a, c, cid)
		for _, p := range pays {
			if p.PayNo == c.Param("payNo") {
				period := "余额充值"
				if p.BillID != 0 {
					period = periodByBill[p.BillID]
				}
				respond(c, apitypes.CodeOK, gin.H{
					"receiptNo": "OR-" + p.PayNo, "amount": p.Amount,
					"period": period, "payMethod": p.Method, "status": p.Status,
					"paidAt": time.Now().Format(time.RFC3339), "payNo": p.PayNo,
				})
				return
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
		// 充值也落缴费流水(bill_id NULL + customer_id 归属),否则 /payments 与凭证端点查不到。
		if _, err := a.Billing.CreatePayment(c.Request.Context(), billing.Payment{
			PayNo: payNo, CustomerID: cid, Amount: req.Amount,
			Method: req.PayMethod, Status: "SUCCESS",
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"payNo": payNo, "amount": req.Amount, "payMethod": req.PayMethod, "status": "SUCCESS",
		})
	}
}
