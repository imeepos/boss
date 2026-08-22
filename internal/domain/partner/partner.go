// 招商引资/合作入驻域:企业自助申请(公开) → 后台审核 → 开通企业管理账号。
// 对标 customer/worker onboarding;渠道经销商(CH)域的入驻先行部分。
package partner

import (
	"context"
	"errors"
	"time"
)

// 入驻申请状态枚举(terms.md 通用枚举延伸,对标 customer RegStatus)。
const (
	StatusPending  = "PENDING"  // 待审核
	StatusApproved = "APPROVED" // 已通过(已建子公司 + 管理账号)
	StatusRejected = "REJECTED" // 已驳回
)

// 域错误。
var (
	ErrApplicationNotFound  = errors.New("partner: application not found")
	ErrApplicationConflict  = errors.New("partner: application status conflict") // 非 PENDING 重复审核
	ErrApplicationDuplicate = errors.New("partner: duplicate pending application")
	ErrNotPartner           = errors.New("partner: account is not a partner member")
	ErrStaffNotFound        = errors.New("partner: staff not found")
	ErrStaffScope           = errors.New("partner: staff outside own legal entity")
)

// Application 入驻申请(审核前不入 legal_entities/accounts)。
type Application struct {
	ID                int64      `json:"id"`
	CompanyName       string     `json:"companyName"`
	CreditCode        string     `json:"creditCode"`
	ContactName       string     `json:"contactName"`
	ContactPhone      string     `json:"contactPhone"`
	Email             string     `json:"email"`
	BusinessDesc      string     `json:"businessDesc"`
	Status            string     `json:"status"`
	ReviewNote        string     `json:"reviewNote"`
	ReviewerAccountID int64      `json:"reviewerAccountId"`
	LegalEntityID     int64      `json:"legalEntityId"`  // 0=未开通
	AdminAccountID    int64      `json:"adminAccountId"` // 0=未开通
	SubmittedAt       time.Time  `json:"submittedAt"`
	ReviewedAt        *time.Time `json:"reviewedAt,omitempty"`
}

// ApproveResult 审核通过结果:初始口令仅此一次返回,由审核人线下传达。
type ApproveResult struct {
	ApplicationID   int64  `json:"applicationId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	AdminAccountID  int64  `json:"adminAccountId"`
	Username        string `json:"username"`
	InitialPassword string `json:"initialPassword"`
}

// PartnerProfile 入驻企业档案(工作台首页)。
type PartnerProfile struct {
	LegalEntityID int64     `json:"legalEntityId"`
	CompanyName   string    `json:"companyName"`
	CreditCode    string    `json:"creditCode"`
	ContactName   string    `json:"contactName"`
	ContactPhone  string    `json:"contactPhone"`
	Email         string    `json:"email"`
	AppliedAt     time.Time `json:"appliedAt"`
	ApprovedAt    time.Time `json:"approvedAt"`
}

// StaffRow 企业员工账号(accounts 按 legal_entity_id 归属,角色限 partner_*)。
type StaffRow struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	RealName  string    `json:"realName"`
	Phone     string    `json:"phone"`
	RoleCode  string    `json:"roleCode"`
	Status    int16     `json:"status"` // 1启用 0停用
	CreatedAt time.Time `json:"createdAt"`
}

// OrderRow 企业订单(orders 按 legal_entity_id 隔离,只读视图)。
type OrderRow struct {
	ID           int64     `json:"id"`
	OrderNo      string    `json:"orderNo"`
	CustomerName string    `json:"customerName"`
	Stage        int8      `json:"stage"`  // 1~12(terms.md)
	Status       string    `json:"status"` // PENDING/RESERVED/INSTALLING/DONE/CANCELLED
	CreatedAt    time.Time `json:"createdAt"`
}

// Service 入驻域服务口。
type Service interface {
	// Submit 企业自助提交入驻申请,落 PENDING;同信用码存在未审申请则拒。
	Submit(ctx context.Context, app Application) (int64, error)
	// ListApplications 按状态(空=全部)列出申请,提交时间倒序。
	ListApplications(ctx context.Context, status string) ([]Application, error)
	// Approve 审核通过:建 legal_entities + partner_admin 账号,回填两个 id。
	// 初始口令随机生成,仅在返回值出现一次。
	Approve(ctx context.Context, id, reviewerAccountID int64) (ApproveResult, error)
	// Reject 审核驳回:记审核意见(仅作用于 PENDING)。
	Reject(ctx context.Context, id, reviewerAccountID int64, note string) error

	// Profile 入驻企业档案(按管理员/员工账号归属的 legal_entity)。
	Profile(ctx context.Context, accountID int64) (PartnerProfile, error)
	// ListStaff 本企业员工账号(id 升序)。
	ListStaff(ctx context.Context, accountID int64) ([]StaffRow, error)
	// CreateStaff 企业管理员新建员工账号(角色 partner_staff,归属本企业)。
	CreateStaff(ctx context.Context, adminAccountID int64, username, password, realName, phone string) (int64, error)
	// SetStaffStatus 启用/停用本企业员工(越权拒)。
	SetStaffStatus(ctx context.Context, adminAccountID, staffID int64, status int16) error
	// ListOrders 本企业订单(创建时间倒序)。
	ListOrders(ctx context.Context, accountID int64) ([]OrderRow, error)
}
