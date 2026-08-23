package userapi

// 用户端门户 Billing 域:账单明细/缴费凭证/余额充值/申请开票 + 门户偏好存取。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
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

// portalPaymentPeriod 缴费流水的账期文案(无账单归属视为余额充值)。
func portalPaymentPeriod(p billing.Payment, periodByBill map[int64]string) string {
	if p.BillID == 0 {
		return "余额充值"
	}
	return periodByBill[p.BillID]
}
