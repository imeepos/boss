package app

import (
	"context"
	"time"

	"github.com/ymm-001/boss/internal/domain/billing"
)

// RunDunning 欠费催收批处理(Q3 收口):逾期标记 → 欠费快照 → 超停机线自动停机。
// graceDays 宽限天数(账单生成起算);stopAfterDays 最早逾期超此天数即停机。
// 单客户失败不中断批次;停机动作为尽力而为,任务留痕 FAILED 可经 retry 端点重放。
func (a *Application) RunDunning(ctx context.Context, graceDays, stopAfterDays int) (billing.DunningResult, error) {
	now := time.Now().UTC()
	res := billing.DunningResult{StoppedIDs: []int64{}}
	if a.Dunning == nil {
		return res, nil
	}
	n, err := a.Dunning.MarkOverdueBills(ctx, graceDays)
	if err != nil {
		return res, err
	}
	res.OverdueBills = n

	customers, err := a.Dunning.ListOverdueCustomers(ctx)
	if err != nil {
		return res, err
	}
	for _, oc := range customers {
		days := billing.ArrearsDaysFrom(now, oc.OldestAt)
		status := billing.ArrearsCollecting
		if int(days) >= stopAfterDays {
			status = billing.ArrearsStopped
		}
		if a.Arrears != nil {
			if _, err := a.Arrears.UpsertArrears(ctx, billing.Arrears{
				CustomerID: oc.CustomerID, Amount: oc.Amount, Days: days, Status: status,
			}); err != nil {
				continue
			}
		}
		res.ArrearsUps++
		if status == billing.ArrearsStopped {
			if a.stopCustomer(ctx, oc.CustomerID) {
				res.StoppedIDs = append(res.StoppedIDs, oc.CustomerID)
			}
		}
	}
	return res, nil
}

// stopCustomer 停机:LO ACTIVE→SUSPENDED + STOP 流水留痕;返回是否实际执行。
func (a *Application) stopCustomer(ctx context.Context, customerID int64) bool {
	if a.Aaa == nil {
		return false
	}
	lo, err := a.Aaa.GetLoAccountByCustomer(ctx, customerID)
	if err != nil || lo == nil || lo.Status != "ACTIVE" {
		return false
	}
	status := "DONE"
	if err := a.Aaa.SuspendLoAccount(ctx, lo.ID); err != nil {
		status = "FAILED"
	}
	if a.Arrears != nil {
		_, _ = a.Arrears.AppendStopResumeTask(ctx, billing.StopResumeTask{
			CustomerID: customerID, LoAccountID: lo.ID, Action: "STOP", Status: status,
		})
	}
	return true
}
