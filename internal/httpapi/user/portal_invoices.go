package userapi

// 用户端门户发票域:电子发票聚合(开票信息 + 可开票账期 + 记录)与开票申请。

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pdfgen"
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
						"invoiceNo": inv.InvoiceNo,
						"period":    inv.BillNo, "amount": inv.TotalAmount,
						"issuedAt": inv.IssuedAt, "pdfUrl": "/api/user/v1/invoices/" + inv.InvoiceNo + "/pdf",
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

// portalApplyInvoice POST /invoices:申请开票(已缴账单才可开票)。
// 真正落 billing invoices 表;bill 不存在/非本人返回 CodeNotFound,重复申请幂等返回已有票。
func portalApplyInvoice(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			BillNo string `json:"billNo" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if a.Tax == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		inv, err := a.Tax.IssueInvoiceForBill(c.Request.Context(), cid, req.BillNo)
		if errors.Is(err, billing.ErrNotFound) {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "invoiceNo": inv.InvoiceNo, "invoiceId": inv.ID})
	}
}

// portalInvoicePdf GET /invoices/:invoiceNo/pdf:电子发票 PDF(税域发票按号寻址,归属过滤)。
func portalInvoicePdf(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if a.Tax == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		invs, err := a.Tax.ListInvoices(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		for _, inv := range invs {
			if inv.InvoiceNo != c.Param("invoiceNo") {
				continue
			}
			pdf := pdfgen.Build("BOSS 电子发票("+inv.InvoiceNo+")", []string{
				"发票号: " + inv.InvoiceNo,
				"账单号: " + inv.BillNo,
				"抬头: " + inv.Title,
				fmt.Sprintf("净额: %.2f 元", inv.NetAmount),
				fmt.Sprintf("税额(%.0f%%): %.2f 元", inv.VatRate*100, inv.VatAmount),
				fmt.Sprintf("价税合计: %.2f 元", inv.TotalAmount),
				"状态: " + inv.Status,
				"税局票号: " + inv.TaxNo,
				"开票时间: " + inv.IssuedAt.In(clock.Location()).Format("2006-01-02 15:04:05"),
				"", "系统发票记录无法定效力,以税局回执为准(tax_no)。",
			})
			portalServePdf(c, "invoice-"+inv.InvoiceNo+".pdf", pdf)
			return
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}
