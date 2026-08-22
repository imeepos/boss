package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
)

// registerBillingRoutes 注册计费账务域路由(承接 api/openapi/admin/billing.yaml)。
func registerBillingRoutes(g *gin.RouterGroup, a *app.Application) {
	b := g.Group("/bills", requirePerm(a.User, "menu:billing"))
	b.GET("", listBills(a))
	g.GET("/payments", requirePerm(a.User, "menu:payment"), listPayments(a))
	// 全额退款(000112):流水 REFUNDED 留痕 + 账单回 UNPAID;发票不自动作废。
	g.POST("/payments/:id/refund", requirePerm(a.User, "menu:payment"), refundPayment(a))
	g.GET("/arrears", requirePerm(a.User, "menu:arrears"), listArrears(a))
	g.GET("/stop-resume-tasks", requirePerm(a.User, "menu:stopsrv"), listStopResumeTasks(a))
	// 欠费停机/复机:为客户生成停复机任务(经其 1:1 LO 账号)。网络侧执行在阶段7。
	stop := g.Group("/arrears", requirePerm(a.User, "menu:stopsrv"))
	stop.POST("/:customerId/stop", appendCustomerStop(a))
	stop.POST("/:customerId/resume", appendCustomerResume(a))
	// 失败停复机任务重试:重放 LO 账号状态迁移,结果回写任务。
	g.POST("/stop-resume-tasks/:taskId/retry", requirePerm(a.User, "menu:stopsrv"), retryStopResumeTask(a))
	// 缴费渠道对账批次列表 + 差异挂起批次平账(billing.yaml /reconciliations)。
	g.GET("/reconciliations", requirePerm(a.User, "menu:paycheck"), listReconciliations(a))
	g.POST("/reconciliations/:batchNo/settle", requirePerm(a.User, "menu:paycheck"), settleReconciliation(a))
	// 渠道对账行级明细(D6 差异定位):批次下逐行 items,差异种类见 diffKind。
	g.GET("/reconciliations/:batchNo/items", requirePerm(a.User, "menu:paycheck"), listReconciliationItems(a))
	// 渠道侧流水按行录入并自动比对生成 items(本期手工录入,自动拉流水不在范围)。
	g.POST("/reconciliations/:batchNo/statement", requirePerm(a.User, "menu:paycheck"), recordChannelStatement(a))
	// 自动对账:按渠道建当日批次(幂等)并从已配置源拉流水比对;manual 渠道只建批。
	g.POST("/reconciliations/auto", requirePerm(a.User, "menu:paycheck"), autoReconcile(a))
	// 账实核对:应收/实收/开票三角,按账单定位差异(与渠道对账正交)。
	g.GET("/billing/ledger-recon", requirePerm(a.User, "menu:paycheck"), ledgerReconHandler(a))
}

// execStopResume 对 LO 账号执行停/复机迁移,返回任务落账状态(DONE/FAILED)。
func execStopResume(a *app.Application, c *gin.Context, loID int64, action string) string {
	transit := a.Aaa.ResumeLoAccount
	if action == "STOP" {
		transit = a.Aaa.SuspendLoAccount
	}
	if err := transit(c.Request.Context(), loID); err != nil {
		return "FAILED"
	}
	return "DONE"
}

// appendStopResume 为客户生成停复机任务:查 LO 账号 → 追加任务(动作 STOP/RESUME)。
func appendStopResume(a *app.Application, c *gin.Context, customerID int64, action string) error {
	lo, err := a.Aaa.GetLoAccountByCustomer(c.Request.Context(), customerID)
	if err != nil {
		return err
	}
	// W6 停复机即时生效:任务留痕 + LO 账号状态原子迁移(停机在线无网/缴费即恢复)。
	taskStatus := execStopResume(a, c, lo.ID, action)
	_, err = a.Arrears.AppendStopResumeTask(c.Request.Context(), billing.StopResumeTask{
		CustomerID: customerID, LoAccountID: lo.ID, Action: action, Status: taskStatus,
	})
	return err
}
