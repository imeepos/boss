package userapi

// 用户端门户发票域:电子发票聚合(开票信息 + 可开票账期 + 记录)与开票申请。

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pdfgen"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalListInvoices GET /invoices:电子发票聚合(开票信息 + 可开票账期 + 记录)。
func portalListInvoices(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		periods, ok := portalInvoiceAvailablePeriods(c, a, cid)
		if !ok {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"titleType":        "个人",
			"title":            portalInvoiceTitle(c, a, cid),
			"taxNo":            "",
			"availablePeriods": periods,
			"records":          portalInvoiceRecords(c, a, cid),
		})
	}
}

// portalInvoiceTitle 客户名作为开票抬头(customers 主档取不到 → 空串)。
func portalInvoiceTitle(c *gin.Context, a *app.Application, cid int64) string {
	if cust, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
		return cust.Name
	}
	return ""
}

// portalInvoiceAvailablePeriods 已缴账期作为可开票账期(失败已 respond → false;成功 → periods, true)。
func portalInvoiceAvailablePeriods(c *gin.Context, a *app.Application, cid int64) ([]gin.H, bool) {
	bills, err := a.Billing.ListBills(c.Request.Context(), cid)
	if err != nil {
		respondErr(c, err)
		return nil, false
	}
	product := planProductName(a, c, cid)
	out := make([]gin.H, 0)
	for _, b := range bills {
		if b.Status == "PAID" {
			out = append(out, portalBill(b, product))
		}
	}
	return out, true
}

// portalInvoiceRecords 已开电子发票记录(税域未接入或失败 → 空,主流程不报错)。
func portalInvoiceRecords(c *gin.Context, a *app.Application, cid int64) []gin.H {
	if a.Tax == nil {
		return []gin.H{}
	}
	invs, err := a.Tax.ListInvoices(c.Request.Context(), cid)
	if err != nil {
		return []gin.H{}
	}
	out := make([]gin.H, 0, len(invs))
	for _, inv := range invs {
		out = append(out, gin.H{
			"invoiceNo": inv.InvoiceNo,
			"period":    inv.BillNo, "amount": inv.TotalAmount,
			"issuedAt": inv.IssuedAt, "pdfUrl": "/api/user/v1/invoices/" + inv.InvoiceNo + "/pdf",
		})
	}
	return out
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
			portalServePdf(c, "invoice-"+inv.InvoiceNo+".pdf", buildInvoicePdf(inv))
			return
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// buildInvoicePdf 单张电子发票的 PDF 内容(发票要素 + 免责说明)。
func buildInvoicePdf(inv billing.Invoice) []byte {
	return pdfgen.Build("BOSS 电子发票("+inv.InvoiceNo+")", []string{
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
}
