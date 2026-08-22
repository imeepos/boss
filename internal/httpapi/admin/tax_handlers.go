package adminapi

// 发票税务域 handler 实现(从 tax.go 抽出,registerTaxRoutes 只剩扁平路由表)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// listInvoices 发票列表。
func listInvoices(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Tax.ListInvoices(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// runBilling 出账:批量生成账单 + 自动开票。
func runBilling(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Period string `json:"period" binding:"required"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		n, err := a.Billing.GenerateBills(c.Request.Context(), body.Period)
		if err != nil {
			respondErr(c, err)
			return
		}
		run, err := a.Tax.IssueInvoicesForPeriod(c.Request.Context(), body.Period)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "billing.run", "period", body.Period, gin.H{"bills": n, "issued": run.Issued})
		emitTask(c.Request.Context(), a, refBilling, "run-"+body.Period,
			"出账完成:"+body.Period+" "+strconv.Itoa(n)+" 张账单/"+strconv.Itoa(run.Issued)+" 张发票", linkBilling, false)
		respond(c, apitypes.CodeOK, gin.H{"bills": n, "invoices": run})
	}
}

// recordPayment 收款:缴费流水落账 + 账单置 PAID。
func recordPayment(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p billing.Payment
		if !httpx.BindBody(c, &p) {
			return
		}
		id, err := a.Billing.RecordPayment(c.Request.Context(), p)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "payment.record", "payment", p.PayNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// voidInvoice 发票作废。
func voidInvoice(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		var body struct {
			Reason string `json:"reason" binding:"required"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.Tax.VoidInvoice(c.Request.Context(), id, body.Reason); err != nil {
			respondErr(c, err)
			return
		}
		appendTaxTrail(a, c, id, billing.TaxEventVoid, invTaxStatusForTrail(a, c, id), "", "")
		httpx.RecordAudit(a, c, "invoice.void", "invoice", c.Param("id"), gin.H{"reason": body.Reason})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// reissueInvoice 重开:原票 VOID 保留编号 + 新票新 ARN。
func reissueInvoice(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		inv, err := a.Tax.ReissueInvoice(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		appendTaxTrail(a, c, id, billing.TaxEventReissue, "", "", "")
		httpx.RecordAudit(a, c, "invoice.reissue", "invoice", c.Param("id"), gin.H{"newNo": inv.InvoiceNo})
		respond(c, apitypes.CodeOK, gin.H{"invoice": inv})
	}
}

// submitInvoiceToTax 税局网关提交:按发票属地取网关开具并落回执。
func submitInvoiceToTax(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		inv, err := a.Tax.GetInvoice(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		gw, ok2 := invoiceTaxable(c, a, inv)
		if !ok2 {
			return
		}
		receipt, err := gw.Issue(c.Request.Context(), *inv)
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Tax.MarkTaxResult(c.Request.Context(), id, receipt); err != nil {
			respondErr(c, err)
			return
		}
		appendTaxTrail(a, c, id, billing.TaxEventReceipt, receipt.Status, receipt.TaxNo, receipt.FailReason)
		httpx.RecordAudit(a, c, "invoice.taxSubmit", "invoice", inv.InvoiceNo, gin.H{"status": receipt.Status})
		respond(c, apitypes.CodeOK, gin.H{"receipt": receipt})
	}
}

// backfillInvoiceTaxNo 人工通道回填:登记税局票号。
func backfillInvoiceTaxNo(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		var body struct {
			TaxNo string `json:"taxNo" binding:"required"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.Tax.BackfillTaxNo(c.Request.Context(), id, body.TaxNo); err != nil {
			respondErr(c, err)
			return
		}
		appendTaxTrail(a, c, id, billing.TaxEventBackfill, billing.TaxStatusIssued, body.TaxNo, "")
		httpx.RecordAudit(a, c, "invoice.taxBackfill", "invoice", c.Param("id"), gin.H{"taxNo": body.TaxNo})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// listInvoiceTaxEvents GET /invoices/:id/tax-events:发票税局轨迹回放(时间正序)。
func listInvoiceTaxEvents(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		svc, ok2 := a.Tax.(billing.TaxEventService)
		if !ok2 {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		events, err := svc.ListTaxEvents(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": events})
	}
}

// appendTaxTrail 状态迁移成功后落轨迹(best-effort:失败不回滚业务动作,
// 终态以 invoices 列列为准,审计日志双轨兜底;取舍见 docs/design/q3-tax-trail.md)。
func appendTaxTrail(a *app.Application, c *gin.Context, invoiceID int64, event, statusAfter, taxNo, failReason string) {
	svc, ok := a.Tax.(billing.TaxEventService)
	if !ok {
		return
	}
	_, _ = svc.AppendTaxEvent(c.Request.Context(), billing.TaxEvent{
		InvoiceID: invoiceID, Event: event, TaxStatusAfter: statusAfter,
		TaxNo: taxNo, FailReason: failReason, OperatorAccountID: httpx.ClaimsAccountID(c),
	})
}

// invTaxStatusForTrail 作废/重开后回读当时税局状态(VOID/REISSUE 事件携带,非税状态迁移)。
func invTaxStatusForTrail(a *app.Application, c *gin.Context, id int64) string {
	inv, err := a.Tax.GetInvoice(c.Request.Context(), id)
	if err != nil {
		return ""
	}
	return inv.TaxStatus
}

// invoiceTaxable 开票资格:税局网关已配置(非人工)且发票 ISSUED 未开税;
// 失败已回写响应。
func invoiceTaxable(c *gin.Context, a *app.Application, inv *billing.Invoice) (billing.TaxGateway, bool) {
	gw := a.TaxGateway.Get(inv.TaxJurisdiction)
	if gw == nil || gw.Channel() == billing.TaxChannelManual {
		respond(c, apitypes.CodeInvalidParam, gin.H{"hint": "tax gateway not configured; use tax-backfill"})
		return nil, false
	}
	if inv.Status != "ISSUED" || inv.TaxStatus == billing.TaxStatusIssued {
		respondErr(c, billing.ErrInvoiceNotTaxable)
		return nil, false
	}
	return gw, true
}
