package adminapi

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerBillingRoutes 注册计费账务域路由(承接 api/openapi/admin/billing.yaml)。
func registerBillingRoutes(g *gin.RouterGroup, a *app.Application) {
	b := g.Group("/bills", requirePerm(a.User, "menu:billing"))

	b.GET("", func(c *gin.Context) {
		list, err := a.Billing.ListBills(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/payments", requirePerm(a.User, "menu:payment"), func(c *gin.Context) {
		list, err := a.Billing.ListPayments(c.Request.Context(), queryInt64(c, "billId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/arrears", requirePerm(a.User, "menu:arrears"), func(c *gin.Context) {
		list, err := a.Arrears.ListArrears(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/stop-resume-tasks", requirePerm(a.User, "menu:stopsrv"), func(c *gin.Context) {
		list, err := a.Arrears.ListStopResumeTasks(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	// 欠费停机/复机:为客户生成停复机任务(经其 1:1 LO 账号)。网络侧执行在阶段7。
	stop := g.Group("/arrears", requirePerm(a.User, "menu:stopsrv"))
	stop.POST("/:customerId/stop", func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "customerId")
		if !ok {
			return
		}
		if err := appendStopResume(a, c, customerID, "STOP"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	stop.POST("/:customerId/resume", func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "customerId")
		if !ok {
			return
		}
		if err := appendStopResume(a, c, customerID, "RESUME"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 失败停复机任务重试:重放 LO 账号状态迁移,结果回写任务。
	g.POST("/stop-resume-tasks/:taskId/retry", requirePerm(a.User, "menu:stopsrv"), func(c *gin.Context) {
		taskID, ok := httpx.ParsePathParamInt64(c, "taskId")
		if !ok {
			return
		}
		task, err := a.Arrears.GetStopResumeTask(c.Request.Context(), taskID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if task.Status != "FAILED" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		status := execStopResume(a, c, task.LoAccountID, task.Action)
		if err := a.Arrears.UpdateStopResumeStatus(c.Request.Context(), taskID, status); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 缴费渠道对账批次列表 + 差异挂起批次平账(billing.yaml /reconciliations)。
	g.GET("/reconciliations", requirePerm(a.User, "menu:paycheck"), func(c *gin.Context) {
		list, err := a.Recon.ListReconciliations(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.POST("/reconciliations/:batchNo/settle", requirePerm(a.User, "menu:paycheck"), func(c *gin.Context) {
		batchNo := c.Param("batchNo")
		if batchNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "batchNo is required"})
			return
		}
		if err := a.Recon.SettleReconciliation(c.Request.Context(), batchNo); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "reconciliation.settle", "reconciliation", batchNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 渠道对账行级明细(D6 差异定位):批次下逐行 items,差异种类见 diffKind。
	g.GET("/reconciliations/:batchNo/items", requirePerm(a.User, "menu:paycheck"), func(c *gin.Context) {
		batchNo := c.Param("batchNo")
		if batchNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "batchNo is required"})
			return
		}
		b, err := a.Recon.GetReconciliation(c.Request.Context(), batchNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		items, err := a.Recon.ListReconciliationItems(c.Request.Context(), b.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	})

	// 渠道侧流水按行录入并自动比对生成 items(本期手工录入,自动拉流水不在范围)。
	g.POST("/reconciliations/:batchNo/statement", requirePerm(a.User, "menu:paycheck"), func(c *gin.Context) {
		batchNo := c.Param("batchNo")
		if batchNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "batchNo is required"})
			return
		}
		b, err := a.Recon.GetReconciliation(c.Request.Context(), batchNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		var body struct {
			Rows []billing.ChannelStatementRow `json:"rows" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		if err := a.Recon.RecordChannelStatement(c.Request.Context(), b.ID, body.Rows); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "reconciliation.statement", "reconciliation", batchNo, nil)
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})

	// 自动对账:按渠道建当日批次(幂等)并从已配置源拉流水比对;manual 渠道只建批。
	g.POST("/reconciliations/auto", requirePerm(a.User, "menu:paycheck"), func(c *gin.Context) {
		if a.ReconAuto == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		date := time.Now()
		if d := c.Query("date"); d != "" {
			parsed, err := time.ParseInLocation("2006-01-02", d, time.Local)
			if err != nil {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
			date = parsed
		}
		channels := []string{"微信", "支付宝", "线下营业厅"} // 默认渠道目录(000035 注释口径)
		if cs := c.Query("channels"); cs != "" {
			channels = strings.Split(cs, ",")
		}
		results, err := a.ReconAuto.AutoReconcile(c.Request.Context(), date, channels)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "reconciliation.auto", "reconciliation", date.Format("2006-01-02"), nil)
		respond(c, apitypes.CodeOK, gin.H{"date": date.Format("2006-01-02"), "items": results})
	})
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
