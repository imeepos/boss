package customer

import (
	"context"
	"time"
)

// RealNameVerification 客户实名核验记录(核验方式/时间/结果审计轨迹)。
type RealNameVerification struct {
	ID                int64     `json:"id"`
	CustomerID        int64     `json:"customerId"`
	Method            string    `json:"method"` // 人脸/证件OCR/人工/第三方
	VerifiedAt        time.Time `json:"verifiedAt"`
	Result            string    `json:"result"`            // PASS通过/FAIL不通过
	RejectReason      string    `json:"rejectReason"`      // FAIL 时的驳回原因
	OperatorAccountID int64     `json:"operatorAccountId"` // 0=空
	OperatorName      string    `json:"operatorName"`      // 操作人姓名快照
}

// RealNameService 实名核验域服务口(阶段2 + onboarding 实名闭环延伸)。
type RealNameService interface {
	ListVerifications(ctx context.Context, customerID int64) ([]RealNameVerification, error)
	AppendVerification(ctx context.Context, v RealNameVerification) (int64, error)
	// SubmitRealName 提交实名核验资料,落 PENDING(覆盖该客户当前 PENDING 记录)。
	SubmitRealName(ctx context.Context, v CustomerRealNameVerification) (int64, error)
	// GetLatest 取客户当前实名核验(最新一条)。
	GetLatest(ctx context.Context, customerID int64) (*CustomerRealNameVerification, error)
	// Verify 后台核验:结果 PENDING→PASS/FAIL(幂等仅作用于 PENDING;PASS 同步 customers.real_name_status;FAIL 记 reason)。
	Verify(ctx context.Context, customerID int64, result, reason, operatorName string, operatorAccountID int64) error
}
