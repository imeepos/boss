package adminapi

// 工程应付台账路由(P-INFRA-1 W6,审查 F8,迁移 000219;契约 admin/odn.yaml)。
// 应付由 SETTLED 结算单自动生成(见 pg_settlement.go),本组仅承载台账读与登记:
// 付款(部分付款/分期逐笔)/核减(原因必填留痕)/发票登记;全部写操作入审计,
// 失败路径留 [odn-payable] 可 grep 日志(域层)。权限 menu:payables(billing 分组呈现)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerODNPayableRoutes 注册应付台账路由(menu:payables 门禁,先例 permits)。
func registerODNPayableRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:payables")
	g.GET("/odn/payables", perm, odnListPayablesHandler(a))
	g.GET("/odn/payables/:id", perm, odnGetPayableHandler(a))
	g.POST("/odn/payables/:id/payments", perm, odnRegisterPaymentHandler(a))
	g.POST("/odn/payables/:id/deductions", perm, odnDeductPayableHandler(a))
	g.POST("/odn/payables/:id/invoices", perm, odnRegisterInvoiceHandler(a))
}

// odnListPayablesHandler GET /odn/payables:台账列表(status/projectId 过滤)。
func odnListPayablesHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, _ := strconv.ParseInt(c.Query("projectId"), 10, 64)
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		list, err := a.ODN.ListPayables(c.Request.Context(), c.Query("status"), projectID, limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

// odnGetPayableHandler GET /odn/payables/{id}:详情聚合(单头+付款/核减/发票流水)。
func odnGetPayableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		d, err := a.ODN.GetPayable(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		if d == nil {
			respond(c, apitypes.CodeNotFound, gin.H{"reason": "payable not found"})
			return
		}
		respond(c, apitypes.CodeOK, d)
	}
}

// odnPaymentReq 付款登记请求体(金额>0,方式枚举,paidAt 可空=当前时刻)。
type odnPaymentReq struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Method    string  `json:"method" binding:"required,oneof=TRANSFER CASH CHEQUE OTHER"`
	PaidAt    string  `json:"paidAt" binding:"omitempty,len=19|len=10"`
	Reference string  `json:"reference" binding:"omitempty,max=64"`
	Note      string  `json:"note" binding:"omitempty,max=255"`
}

// odnRegisterPaymentHandler POST /odn/payables/{id}/payments:付款流水登记(不超未付余额)。
func odnRegisterPaymentHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnPaymentReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		operator := httpx.ClaimsAccountID(c)
		p, err := a.ODN.RegisterPayablePayment(c.Request.Context(), id, operator,
			req.Amount, req.Method, req.PaidAt, req.Reference, req.Note)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.payable.payment", "construction_payable_payments",
			strconv.FormatInt(p.ID, 10), map[string]any{"payableId": id, "amount": req.Amount,
				"method": req.Method, "paymentNo": p.PaymentNo})
		respond(c, apitypes.CodeOK, p)
	}
}

// odnDeductionReq 核减请求体(原因必填留痕;amount>0)。
type odnDeductionReq struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason string  `json:"reason" binding:"required,max=255"`
}

// odnDeductPayableHandler POST /odn/payables/{id}/deductions:核减(净应付同步,防超付)。
func odnDeductPayableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnDeductionReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		operator := httpx.ClaimsAccountID(c)
		d, err := a.ODN.DeductPayable(c.Request.Context(), id, operator, req.Amount, req.Reason)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.payable.deduct", "construction_payable_deductions",
			strconv.FormatInt(d.ID, 10), map[string]any{"payableId": id, "amount": req.Amount, "reason": req.Reason})
		respond(c, apitypes.CodeOK, d)
	}
}

// odnInvoiceReq 发票登记请求体(发票号必填,同应付内唯一;纯登记不联动税局)。
type odnInvoiceReq struct {
	InvoiceNo  string  `json:"invoiceNo" binding:"required,max=64"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	InvoicedAt string  `json:"invoicedAt" binding:"omitempty,len=10"`
	Note       string  `json:"note" binding:"omitempty,max=255"`
}

// odnRegisterInvoiceHandler POST /odn/payables/{id}/invoices:发票登记。
func odnRegisterInvoiceHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		var req odnInvoiceReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		operator := httpx.ClaimsAccountID(c)
		inv, err := a.ODN.RegisterPayableInvoice(c.Request.Context(), id, operator,
			req.Amount, req.InvoiceNo, req.InvoicedAt, req.Note)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.payable.invoice", "construction_payable_invoices",
			strconv.FormatInt(inv.ID, 10), map[string]any{"payableId": id,
				"invoiceNo": req.InvoiceNo, "amount": req.Amount})
		respond(c, apitypes.CodeOK, inv)
	}
}
