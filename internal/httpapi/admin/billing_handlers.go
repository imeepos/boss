package adminapi

// 计费账务域 handler 实现(从 billing.go 抽出,registerBillingRoutes 只剩扁平路由表)。

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// listBills 客户账单列表。
func listBills(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Billing.ListBills(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// listPayments 缴费流水列表。
func listPayments(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Billing.ListPayments(c.Request.Context(), queryInt64(c, "billId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// refundPayment 全额退款(000112):流水 REFUNDED 留痕,账单回 UNPAID 可重收款。
func refundPayment(a *app.Application) gin.HandlerFunc {
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
		p, err := a.Billing.RefundPayment(c.Request.Context(), id, body.Reason)
		if err != nil {
			respondErr(c, err)
			return
		}
		rollbackPaymentPoints(c.Request.Context(), a, p.CustomerID, id)
		httpx.RecordAudit(a, c, "payment.refund", "payment", p.PayNo, gin.H{"reason": body.Reason})
		respond(c, apitypes.CodeOK, gin.H{"payment": p})
	}
}

// rollbackPaymentPoints 退款冲销缴费积分(2028 Q2):按 payment_id 幂等;
// 冲销失败仅记日志(积分不足时下次人工补偿),不回滚退款主流程。
func rollbackPaymentPoints(ctx context.Context, a *app.Application, customerID, paymentID int64) {
	if a.Points == nil {
		return
	}
	if _, err := a.Points.RollbackPayment(ctx, paymentID, customerID); err != nil {
		log.Printf("[loy-rollback] payment %d rollback failed: %v", paymentID, err)
	}
}

// listArrears 欠费清单。
func listArrears(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Arrears.ListArrears(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// listStopResumeTasks 停复机任务列表。
func listStopResumeTasks(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Arrears.ListStopResumeTasks(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// appendCustomerStop 客户停机:生成 STOP 任务。
func appendCustomerStop(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "customerId")
		if !ok {
			return
		}
		if err := appendStopResume(a, c, customerID, "STOP"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// appendCustomerResume 客户复机:生成 RESUME 任务。
func appendCustomerResume(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "customerId")
		if !ok {
			return
		}
		if err := appendStopResume(a, c, customerID, "RESUME"); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// retryStopResumeTask 失败停复机任务重试:重放 LO 账号状态迁移,结果回写任务。
func retryStopResumeTask(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// listReconciliations 对账批次列表。
func listReconciliations(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Recon.ListReconciliations(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// settleReconciliation 差异挂起批次平账。
func settleReconciliation(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// listReconciliationItems 对账行级明细。
func listReconciliationItems(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// recordChannelStatement 渠道侧流水按行录入并自动比对生成 items。
func recordChannelStatement(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// autoReconcile 自动对账:按渠道建当日批次并从已配置源拉流水比对。
func autoReconcile(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.ReconAuto == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		date := clock.Now()
		if d := c.Query("date"); d != "" {
			parsed, err := time.ParseInLocation("2006-01-02", d, clock.Location())
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
	}
}
