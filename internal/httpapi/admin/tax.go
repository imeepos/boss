package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// pathIDValid 解析路径参数为 int64;非法值返回 400。
func pathIDValid(c *gin.Context, name string) (int64, bool) {
	return httpx.ParsePathParamInt64(c, name)
}

// registerTaxRoutes 注册发票税务域路由(TAX/AG-04,入 billing 分组)。
// 链路:收款 POST /payments → 出账+自动开票 POST /billing-runs → 作废/重开 /invoices。
func registerTaxRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/invoices", requirePerm(a.User, "menu:billing"), func(c *gin.Context) {
		list, err := a.Tax.ListInvoices(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 出账:批量生成账单 → 自动开票(CT-007:同账期不重复开票,失败账单返回人工处理)。
	g.POST("/billing-runs", requirePerm(a.User, "menu:billing"), func(c *gin.Context) {
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
		respond(c, apitypes.CodeOK, gin.H{"bills": n, "invoices": run})
	})

	// 收款:缴费流水落账 + 账单置 PAID(同事务);pay_no 唯一幂等。
	g.POST("/payments", requirePerm(a.User, "menu:payment"), func(c *gin.Context) {
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
	})

	g.POST("/invoices/:id/void", requirePerm(a.User, "menu:billing"), func(c *gin.Context) {
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
		httpx.RecordAudit(a, c, "invoice.void", "invoice", c.Param("id"), gin.H{"reason": body.Reason})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 重开:原票 VOID 保留编号 + 新票新 ARN(TAX-003)。
	g.POST("/invoices/:id/reissue", requirePerm(a.User, "menu:billing"), func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		inv, err := a.Tax.ReissueInvoice(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "invoice.reissue", "invoice", c.Param("id"), gin.H{"newNo": inv.InvoiceNo})
		respond(c, apitypes.CodeOK, gin.H{"invoice": inv})
	})

	// 税局网关提交:按发票属地取网关开具并落回执;未注册网关(人工通道)则提示走回填。
	g.POST("/invoices/:id/tax-submit", requirePerm(a.User, "menu:billing"), func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		inv, err := a.Tax.GetInvoice(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		gw := a.TaxGateway.Get(inv.TaxJurisdiction)
		if gw == nil || gw.Channel() == billing.TaxChannelManual {
			respond(c, apitypes.CodeInvalidParam, gin.H{"hint": "tax gateway not configured; use tax-backfill"})
			return
		}
		if inv.Status != "ISSUED" || inv.TaxStatus == billing.TaxStatusIssued {
			respondErr(c, billing.ErrInvoiceNotTaxable)
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
		httpx.RecordAudit(a, c, "invoice.taxSubmit", "invoice", inv.InvoiceNo, gin.H{"status": receipt.Status})
		respond(c, apitypes.CodeOK, gin.H{"receipt": receipt})
	})

	// 人工通道回填:运营者在税局平台(数电票/BIR)开具后登记税局票号。
	g.POST("/invoices/:id/tax-backfill", requirePerm(a.User, "menu:billing"), func(c *gin.Context) {
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
		httpx.RecordAudit(a, c, "invoice.taxBackfill", "invoice", c.Param("id"), gin.H{"taxNo": body.TaxNo})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}
