package userapi

// 门户 Billing 扩展:自动缴费开关(portal_billing_prefs)与缴费凭证 PDF 下载。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pdfgen"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalAutoPayGet GET /billing/auto-pay:自动缴费开通状态。
func portalAutoPayGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		on, err := a.Portal.AutoPay(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"autoPayEnabled": on})
	}
}

// portalAutoPaySet POST /billing/auto-pay:开通/关闭自动缴费。
func portalAutoPaySet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Portal.SetAutoPay(c.Request.Context(), cid, req.Enabled); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "autoPayEnabled": req.Enabled})
	}
}

// portalReceiptPdf GET /payments/:payNo/receipt.pdf:缴费/充值凭证 PDF(按客户聚合寻址)。
func portalReceiptPdf(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		pays, err := a.Billing.ListPaymentsByCustomer(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		periodByBill := portalBillPeriods(a, c, cid)
		for _, p := range pays {
			if p.PayNo != c.Param("payNo") {
				continue
			}
			period := "余额充值"
			if p.BillID != 0 {
				period = periodByBill[p.BillID]
			}
			pdf := pdfgen.Build("BOSS 缴费凭证", []string{
				"凭证号: OR-" + p.PayNo,
				"支付单号: " + p.PayNo,
				fmt.Sprintf("账期: %s", period),
				fmt.Sprintf("金额: %.2f 元", p.Amount),
				"支付方式: " + portalPayMethodLabel[p.Method],
				"状态: " + p.Status,
				"支付时间: " + clock.Now().Format("2006-01-02 15:04:05"),
				"", "本凭证由 BOSS 系统出具,仅供缴费记录查询使用。",
			})
			portalServePdf(c, "receipt-"+p.PayNo+".pdf", pdf)
			return
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

// portalPayMethodLabel 支付方式中文标签。
var portalPayMethodLabel = map[string]string{
	"wechat": "微信支付", "alipay": "支付宝", "card": "银行卡", "cash": "现金", "prepaid": "余额",
}

// portalServePdf 输出 PDF 附件。
func portalServePdf(c *gin.Context, name string, pdf []byte) {
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(200, "application/pdf", pdf)
}
