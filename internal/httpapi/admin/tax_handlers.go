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

// recordPayment 收款:缴费流水落账 + 账单条件置 PAID(同事务,domain 层)。
// 柜面凭证要素:siteName/counterCode 表单录入,operatorName 服务端取登录态;
// method 白名单外 domain 拒收(42200);cash 单笔超限额拒绝并写审计留痕。
func recordPayment(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p billing.Payment
		if !httpx.BindBody(c, &p) {
			return
		}
		p.OperatorName = httpx.ClaimsUsername(c)
		if !enforceCashLimit(c, a, p) {
			return
		}
		// 用带券回执(券码为空即纯收款):回执带兜底生成后的 payNo,响应可回显。
		receipt, err := a.Billing.RecordPaymentWithCoupon(c.Request.Context(), p)
		if err != nil {
			respondErr(c, err)
			return
		}
		p.PayNo = receipt.PayNo
		resumeCustomerAfterPay(c, a, p)
		// 审计补金额/方式/客户(纪要待定项③,郑凯:追责四问必须能答全)。
		httpx.RecordAudit(a, c, "payment.record", "payment", p.PayNo, gin.H{
			"amount": p.Amount, "method": p.Method, "customerId": p.CustomerID,
			"siteName": p.SiteName, "counterCode": p.CounterCode, "operator": p.OperatorName,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": receipt.PaymentID, "payNo": receipt.PayNo})
	}
}

// resumeCustomerAfterPay 缴费归属客户(直传优先,账单兜底)后自动复机(尽力而为)。
func resumeCustomerAfterPay(c *gin.Context, a *app.Application, p billing.Payment) {
	cid := p.CustomerID
	if cid == 0 && p.BillID > 0 {
		if b, err := a.Billing.GetBill(c.Request.Context(), p.BillID); err == nil && b != nil {
			cid = b.CustomerID
		}
	}
	a.ResumeAfterPayment(c.Request.Context(), cid)
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
// MarkTaxResult 负责记录 RECEIPT 轨迹(含重复/乱序幂等),handler 不再重复 append。
func submitInvoiceToTax(a *app.Application) gin.HandlerFunc {
	return submitTax(a, "invoice.taxSubmit")
}

// retryInvoiceTax 税局重试:从 FAILED/BLOCKED 状态重新提交,行为同 tax-submit。
func retryInvoiceTax(a *app.Application) gin.HandlerFunc {
	return submitTax(a, "invoice.taxRetry")
}

// replayInvoiceTax 税局回放:对 SUBMITTED/FAILED/PENDING 重新推动网关,幂等。
func replayInvoiceTax(a *app.Application) gin.HandlerFunc {
	return submitTax(a, "invoice.taxReplay")
}

// submitTax 税局提交/重试/回放共享内核:网关 Issue → MarkTaxResult(含轨迹) → 审计。
// MarkTaxResult 已内嵌 RECEIPT 轨迹,handler 不重复 append。
func submitTax(a *app.Application, audit string) gin.HandlerFunc {
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
		httpx.RecordAudit(a, c, audit, "invoice", inv.InvoiceNo,
			gin.H{"status": receipt.Status, "externalId": receipt.ExternalID})
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

// taxFailureDetails 单票税务失败详情，供运营定位并判断是否可重试。
func taxFailureDetails(a *app.Application) gin.HandlerFunc {
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
		var last *billing.TaxEvent
		if svc, ok := a.Tax.(billing.TaxEventService); ok {
			if events, e := svc.ListTaxEvents(c.Request.Context(), id); e == nil {
				for i := len(events) - 1; i >= 0; i-- {
					if events[i].FailReason != "" {
						last = &events[i]
						break
					}
				}
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"invoice": inv, "lastFailure": last,
			"retryable": inv.TaxStatus == billing.TaxStatusFailed || inv.TaxStatus == billing.TaxStatusBlocked})
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
