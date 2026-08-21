package customer

import (
	"context"
	"errors"
	"time"
)

// 客户注册 / 审核 / 实名认证 子域错误(对标 worker onboarding)。
var (
	ErrRegistrationNotFound = errors.New("customer: registration not found")
	ErrRegistrationConflict = errors.New("customer: registration status conflict") // 非 PENDING 重复审核
	ErrRealNameNotFound     = errors.New("customer: real name verification not found")
	ErrRealNameConflict     = errors.New("customer: real name verification conflict") // 非 PENDING 重复核验
)

// 注册申请状态枚举(terms.md 通用枚举延伸,对标 worker)。
const (
	RegStatusPending  = "PENDING"  // 待审核
	RegStatusApproved = "APPROVED" // 已通过(已建 customers 主档)
	RegStatusRejected = "REJECTED" // 已驳回
)

// 实名核验结果枚举(terms.md result 枚举延伸,对标 worker)。
const (
	RealNamePending = "PENDING" // 待核验
	RealNamePass    = "PASS"    // 通过
	RealNameFail    = "FAIL"    // 不通过
)

// Registration 客户注册申请(对标师傅注册;审核前不入 customers 主档)。
type Registration struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Phone             string     `json:"phone"`
	IDCardNo          string     `json:"idCardNo"`
	LegalEntityID     int64      `json:"legalEntityId"`
	AddressID         int64      `json:"addressId"`
	RegionID          int64      `json:"regionId"`
	Status            string     `json:"status"`
	ReviewNote        string     `json:"reviewNote"`
	ReviewerAccountID int64      `json:"reviewerAccountId"`
	CustomerID        int64      `json:"customerId"` // 0=未建主档
	SubmittedAt       time.Time  `json:"submittedAt"`
	ReviewedAt        *time.Time `json:"reviewedAt,omitempty"`
}

// CustomerRealNameVerification 客户实名核验(与 customers 1:1 当前态)。
type CustomerRealNameVerification struct {
	ID                int64     `json:"id"`
	CustomerID        int64     `json:"customerId"`
	Method            string    `json:"method"`
	RealName          string    `json:"realName"`
	IDCardNo          string    `json:"idCardNo"`
	Result            string    `json:"result"`
	RejectReason      string    `json:"rejectReason"`  // FAIL 时后台/自动核验填写
	IDCardFrontID     int64     `json:"idCardFrontId"` // attachments.id 人像面,0=未传
	IDCardBackID      int64     `json:"idCardBackId"`  // attachments.id 国徽面,0=未传
	VerifiedAt        time.Time `json:"verifiedAt"`
	OperatorAccountID int64     `json:"operatorAccountId"`
	OperatorName      string    `json:"operatorName"`
}

// OnboardingService 客户注册 / 审核 子域服务口(对标 worker.OnboardingService)。
type OnboardingService interface {
	// Submit 客户自助注册,落 PENDING 申请。
	Submit(ctx context.Context, reg Registration) (int64, error)
	// ListRegistrations 按状态(空=全部)列出申请,提交时间倒序。
	ListRegistrations(ctx context.Context, status string) ([]Registration, error)
	// Approve 审核通过:状态 PENDING→APPROVED,并建 customers 主档,回填 customer_id。
	Approve(ctx context.Context, id, reviewerAccountID int64) (customerID int64, err error)
	// Reject 审核驳回:状态 PENDING→REJECTED,记审核意见(幂等仅作用于 PENDING)。
	Reject(ctx context.Context, id, reviewerAccountID int64, note string) error
}
