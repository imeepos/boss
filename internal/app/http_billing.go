package app

import (
	"context"

	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerBillingRoutes 注册计费账务域路由(承接 api/openapi/admin/billing.yaml)。
func registerBillingRoutes(g *gin.RouterGroup, a *Application) {
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
		customerID, _ := strconv.ParseInt(c.Param("customerId"), 10, 64)
		if err := a.appendStopResume(c, customerID, "STOP"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
	stop.POST("/:customerId/resume", func(c *gin.Context) {
		customerID, _ := strconv.ParseInt(c.Param("customerId"), 10, 64)
		if err := a.appendStopResume(c, customerID, "RESUME"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}

// appendStopResume 为客户生成停复机任务:查 LO 账号 → 追加任务(动作 STOP/RESUME)。
func (a *Application) appendStopResume(c *gin.Context, customerID int64, action string) error {
	lo, err := a.Aaa.GetLoAccountByCustomer(c.Request.Context(), customerID)
	if err != nil {
		return err
	}
	// W6 停复机即时生效:任务留痕 + LO 账号状态原子迁移(停机在线无网/缴费即恢复)。
	var transit func(context.Context, int64) error
	taskStatus := "DONE"
	if action == "STOP" {
		transit = a.Aaa.SuspendLoAccount
	} else {
		transit = a.Aaa.ResumeLoAccount
	}
	if err := transit(c.Request.Context(), lo.ID); err != nil {
		taskStatus = "FAILED" // 非法迁移(重复停机等)记录失败任务,不中断响应
	}
	_, err = a.Arrears.AppendStopResumeTask(c.Request.Context(), billing.StopResumeTask{
		CustomerID: customerID, LoAccountID: lo.ID, Action: action, Status: taskStatus,
	})
	return err
}
