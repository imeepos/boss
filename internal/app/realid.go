package app

// 实名二要素自动核验编排:提交后调通道,结论落 verifications 并同步客户主档;
// 通道未启用/调用失败保持 PENDING 走人工核验,不阻塞提交。

import (
	"context"
	"log"

	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/pkg/realid"
)

// RealIDAutoOperator 自动核验落 verifications 的操作人快照(0=空,无后台账号)。
const RealIDAutoOperator = "阿里云二要素"

// AutoVerifyRealName 客户实名提交后自动核验;返回最终 result(PASS/FAIL/PENDING)。
// PENDING 语义:通道未启用(ErrDisabled 由 nil 表达)、通道调用失败或落库失败 → 人工核验兜底。
func (a *Application) AutoVerifyRealName(ctx context.Context, customerID int64, name, idNo string) string {
	if a.RealID == nil {
		return customer.RealNamePending
	}
	decision, err := a.RealID.Verify(ctx, name, idNo)
	if err != nil {
		log.Printf("realid: auto verify failed customer=%d: %v", customerID, err)
		return customer.RealNamePending
	}
	// 二要素 FAIL 即"姓名与证件号不一致",写入驳回原因供用户端驳回页展示。
	reason := ""
	if decision == customer.RealNameFail {
		reason = "姓名与证件号码不一致,请核对后重新提交"
	}
	if err := a.CustomerRealName.Verify(ctx, customerID, decision, reason, RealIDAutoOperator, 0); err != nil {
		log.Printf("realid: persist %s failed customer=%d: %v", decision, customerID, err)
		return customer.RealNamePending
	}
	return decision
}

// compile-time: 接口存在即约束;通道实现见 internal/pkg/realid。
var _ = realid.Verifier(nil)
