package app

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/billing"
)

// ResumeAfterPayment 缴费成功后自动复机(欠费停机收口,Q3):
// 客户 LO 账号处 SUSPENDED 时立即复机并落 RESUME 流水留痕;
// 非 SUSPENDED / 无 LO 账号 = 无需复机,静默跳过。
// 尽力而为:复机失败不回滚缴费,任务留 FAILED 供 /stop-resume-tasks/:id/retry 重试。
func (a *Application) ResumeAfterPayment(ctx context.Context, customerID int64) {
	if customerID == 0 || a.Aaa == nil || a.Arrears == nil {
		return
	}
	lo, err := a.Aaa.GetLoAccountByCustomer(ctx, customerID)
	if err != nil || lo == nil || lo.Status != "SUSPENDED" {
		return
	}
	status := "DONE"
	if err := a.Aaa.ResumeLoAccount(ctx, lo.ID); err != nil {
		status = "FAILED"
	}
	_, _ = a.Arrears.AppendStopResumeTask(ctx, billing.StopResumeTask{
		CustomerID: customerID, LoAccountID: lo.ID, Action: "RESUME", Status: status,
	})
}
