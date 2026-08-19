package worker

import (
	"context"
	"errors"
	"time"
)

// 师傅注册 / 审核 / 实名认证 子域错误。
var (
	ErrRegistrationNotFound = errors.New("worker: registration not found")
	ErrRegistrationConflict = errors.New("worker: registration status conflict") // 非 PENDING 重复审核
	ErrRealNameNotFound     = errors.New("worker: real name verification not found")
	ErrRealNameConflict     = errors.New("worker: real name verification conflict") // 非 PENDING 重复核验
)

// 注册申请状态枚举(terms.md 通用枚举延伸)。
const (
	RegStatusPending  = "PENDING"  // 待审核
	RegStatusApproved = "APPROVED" // 已通过(已建 workers 主档)
	RegStatusRejected = "REJECTED" // 已驳回
)

// 实名核验结果枚举(terms.md result 枚举延伸)。
const (
	RealNamePending = "PENDING" // 待核验
	RealNamePass    = "PASS"    // 通过
	RealNameFail    = "FAIL"    // 不通过
)

// Registration 师傅注册申请(对标客户自助建档,审核前不入 workers 主档)。
type Registration struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Phone             string     `json:"phone"`
	IDCardNo          string     `json:"idCardNo"`
	GroupID           int64      `json:"groupId"`
	RegionID          int64      `json:"regionId"`
	Status            string     `json:"status"`
	ReviewNote        string     `json:"reviewNote"`
	ReviewerAccountID int64      `json:"reviewerAccountId"`
	WorkerID          int64      `json:"workerId"` // 0=未建主档
	SubmittedAt       time.Time  `json:"submittedAt"`
	ReviewedAt        *time.Time `json:"reviewedAt,omitempty"`
}

// WorkerRealNameVerification 师傅实名核验(对标客户 real_name_verifications)。
type WorkerRealNameVerification struct {
	ID                 int64     `json:"id"`
	WorkerID           int64     `json:"workerId"`
	Method             string    `json:"method"`
	RealName           string    `json:"realName"`
	IDCardNo           string    `json:"idCardNo"`
	Result             string    `json:"result"`
	VerifiedAt         time.Time `json:"verifiedAt"`
	OperatorAccountID  int64     `json:"operatorAccountId"`
	OperatorName       string    `json:"operatorName"`
}

// OnboardingService 师傅注册 / 审核 子域服务口。
type OnboardingService interface {
	// Submit 师傅自助注册,落 PENDING 申请。
	Submit(ctx context.Context, reg Registration) (int64, error)
	// ListRegistrations 按状态(空=全部)列出申请,提交时间倒序。
	ListRegistrations(ctx context.Context, status string) ([]Registration, error)
	// Approve 审核通过:状态 PENDING→APPROVED,并建 workers 主档,回填 worker_id。
	Approve(ctx context.Context, id, reviewerAccountID int64) (workerID int64, err error)
	// Reject 审核驳回:状态 PENDING→REJECTED,记审核意见(幂等仅作用于 PENDING)。
	Reject(ctx context.Context, id, reviewerAccountID int64, note string) error
}

// RealNameService 师傅实名核验子域服务口。
type RealNameService interface {
	// SubmitRealName 师傅(或后台代录)提交实名资料,落 PENDING 核验(覆盖该师傅当前 PENDING 记录)。
	SubmitRealName(ctx context.Context, v WorkerRealNameVerification) (int64, error)
	// GetLatest 取师傅当前实名核验(最新一条)。
	GetLatest(ctx context.Context, workerID int64) (*WorkerRealNameVerification, error)
	// Verify 后台核验:结果 PENDING→PASS/FAIL(幂等仅作用于 PENDING)。
	Verify(ctx context.Context, workerID int64, result, operatorName string, operatorAccountID int64) error
}
