package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// pathIDValid 解析路径参数为 int64;非法值返回 400。
func pathIDValid(c *gin.Context, name string) (int64, bool) {
	return httpx.ParsePathParamInt64(c, name)
}

// registerTaxRoutes 注册发票税务域路由(TAX/AG-04,入 billing 分组)。
// 链路:收款 POST /payments → 出账+自动开票 POST /billing-runs → 作废/重开 /invoices。
func registerTaxRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/invoices", requirePerm(a.User, "menu:billing"), listInvoices(a))
	// 出账:批量生成账单 → 自动开票(CT-007:同账期不重复开票,失败账单返回人工处理)。
	g.POST("/billing-runs", requirePerm(a.User, "menu:billing"), runBilling(a))
	// 收款:缴费流水落账 + 账单条件置 PAID(同事务);pay_no 唯一幂等。
	// 柜面现金收款高风险资金入口:独立按钮级权限码(纪要 2026-08-28 Battle A,
	// 与只读看流水 menu:payment、退款 menu:payment 分离,ops 不默认授予)。
	g.POST("/payments", requirePerm(a.User, "menu:payment:cash"), recordPayment(a))
	g.POST("/invoices/:id/void", requirePerm(a.User, "menu:billing"), voidInvoice(a))
	// 重开:原票 VOID 保留编号 + 新票新 ARN(TAX-003)。
	g.POST("/invoices/:id/reissue", requirePerm(a.User, "menu:billing"), reissueInvoice(a))
	// 税局网关提交:按发票属地取网关开具并落回执;未注册网关(人工通道)则提示走回填。
	g.POST("/invoices/:id/tax-submit", requirePerm(a.User, "menu:billing"), submitInvoiceToTax(a))
	g.POST("/invoices/:id/tax-retry", requirePerm(a.User, "menu:billing"), retryInvoiceTax(a))
	g.POST("/invoices/:id/tax-replay", requirePerm(a.User, "menu:billing"), replayInvoiceTax(a))
	g.GET("/invoices/:id/tax-failure", requirePerm(a.User, "menu:billing"), taxFailureDetails(a))
	// 人工通道回填:运营者在税局平台(数电票/BIR)开具后登记税局票号。
	g.POST("/invoices/:id/tax-backfill", requirePerm(a.User, "menu:billing"), backfillInvoiceTaxNo(a))
	// 税局轨迹回放:回执/回填/作废/重开留痕(验收②可追踪)。
	g.GET("/invoices/:id/tax-events", requirePerm(a.User, "menu:billing"), listInvoiceTaxEvents(a))
}
