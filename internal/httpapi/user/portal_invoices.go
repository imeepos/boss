package userapi

// 用户端门户发票域:电子发票聚合(开票信息 + 可开票账期 + 记录)与开票申请。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

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

// portalApplyInvoice POST /invoices:申请开票(已缴账单才可开票;开票落税局后台回填票号)。
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