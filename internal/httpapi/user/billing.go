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
	g.GET("/topups", portalBalanceGet(a))
	g.POST("/topups", portalTopup(a))
	g.GET("/invoices", portalListInvoices(a))
	g.POST("/invoices", portalApplyInvoice(a))
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

// portalListPayments GET /payments:我的缴费记录(跨账单聚合)。
func portalListPayments(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, b := range bills {
			pays, err := a.Billing.ListPayments(c.Request.Context(), b.BillID)
			if err != nil {
				continue
			}
			for _, p := range pays {
				items = append(items, gin.H{
					"payNo": p.PayNo, "amount": p.Amount, "period": b.Period,
					"payMethod": p.Method, "paidAt": time.Now().Format(time.RFC3339),
				})
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// portalListInvoices GET /invoices:电子发票聚合(开票信息 + 可开票账期 + 记录)。
func portalListInvoices(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		title, taxNo := "", ""
		if cust, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
			title = cust.Name
		}
		bills, err := a.Billing.ListBills(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		periods := make([]gin.H, 0)
		for _, b := range bills {
			if b.Status == "PAID" {
				periods = append(periods, portalBill(b, planProductName(a, c, cid)))
			}
		}
		records := make([]gin.H, 0)
		if a.Tax != nil {
			if invs, err := a.Tax.ListInvoices(c.Request.Context(), cid); err == nil {
				for _, inv := range invs {
					records = append(records, gin.H{
						"period": inv.BillNo, "amount": inv.TotalAmount,
						"issuedAt": inv.IssuedAt, "pdfUrl": "",
					})
				}
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"titleType": "个人", "title": title, "taxNo": taxNo,
			"availablePeriods": periods, "records": records,
		})
	}
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
