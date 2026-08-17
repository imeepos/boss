package customer

import (
	"context"
	"time"
)

// RealNameVerification 客户实名核验记录(核验方式/时间/结果审计轨迹)。
type RealNameVerification struct {
	ID                int64
	CustomerID        int64
	Method            string // 人脸/证件OCR/人工/第三方
	VerifiedAt        time.Time
	Result            string // PASS通过/FAIL不通过
	OperatorAccountID int64  // 0=空
	OperatorName      string // 操作人姓名快照
}

// RealNameService 实名核验域服务口(阶段2)。
type RealNameService interface {
	ListVerifications(ctx context.Context, customerID int64) ([]RealNameVerification, error)
	AppendVerification(ctx context.Context, v RealNameVerification) (int64, error)
}
