package odn

import (
	"context"
	"errors"
)

// 工程结算单状态(P-INFRA-1 W1,迁移 000204;枚举登记 terms.md §4)。
const (
	SPending = "PENDING" // 待结算:已发起,应付金额已锁定
	SSettled = "SETTLED" // 已结算:付款确认完成
	SVoided  = "VOIDED"  // 已作废:终态,原单保留历史;项目重开发起以新单表达
)

var (
	// ErrNoContractor 项目未指定承包商不可发起结算(应付对象缺失):40900。
	ErrNoContractor = errors.New("odn: project has no contractor")
	// ErrSettlementState 非法结算单状态转移或项目已锁(存在有效结算单):40900。
	ErrSettlementState = errors.New("odn: invalid settlement status")
	// ErrInvalidInput 结算入参不合法(作废原因缺失等):42200。
	ErrInvalidInput = errors.New("odn: invalid settlement input")
)

// Settlement 工程结算单(应付=发起时 SUM(construction_items.amount);ACCEPTED 后明细锁定不漂移)。
type Settlement struct {
	ID             int64   `json:"id"`
	SettlementNo   string  `json:"settlementNo"`
	ProjectID      int64   `json:"projectId"`
	ProjectNo      string  `json:"projectNo"`
	ContractorID   int64   `json:"contractorId"`
	ContractorName string  `json:"contractorName"`
	TotalAmount    float64 `json:"totalAmount"`
	ItemCount      int64   `json:"itemCount"`
	Status         string  `json:"status"`
	VoidReason     string  `json:"voidReason"`
	CreatedBy      int64   `json:"createdBy"`
	SettledBy      int64   `json:"settledBy"`
	VoidedBy       int64   `json:"voidedBy"`
	CreatedAt      string  `json:"createdAt"`
	SettledAt      string  `json:"settledAt,omitempty"`
	VoidedAt       string  `json:"voidedAt,omitempty"`
}

// ValidateSettlementTransition 结算单状态转移:PENDING→SETTLED;PENDING/SETTLED→VOIDED;
// 同态 no-op;VOIDED 终态。作废必带原因(handler 层校验),重开以新结算单表达。
func ValidateSettlementTransition(from, to string) error {
	if from == to {
		return nil
	}
	switch {
	case from == SPending && to == SSettled:
		return nil
	case from == SPending && to == SVoided:
		return nil
	case from == SSettled && to == SVoided:
		return nil
	default:
		return ErrSettlementState
	}
}

// 结算单存储口(PGStore 实现;接口收敛在 ODNService)。
type SettlementStore interface {
	CreateSettlement(ctx context.Context, projectID, createdBy int64) (*Settlement, error)
	ListSettlements(ctx context.Context, projectID int64) ([]Settlement, error)
	GetSettlement(ctx context.Context, id int64) (*Settlement, error)
	SettleSettlement(ctx context.Context, id, accountID int64) error
	VoidSettlement(ctx context.Context, id, accountID int64, reason string) error
}
