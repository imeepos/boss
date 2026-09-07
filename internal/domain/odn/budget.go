package odn

import (
	"context"
	"errors"
)

// 里程碑状态(P-INFRA-1 W6,迁移 000218;枚举登记 terms.md §4)。
const (
	MPending = "PENDING" // 未完成
	MDone    = "DONE"    // 已完成
)

var (
	// ErrBudgetLocked 预算/里程碑清单编辑被拒(仅项目 PENDING 可改;ACCEPTED 后锁定):40900。
	ErrBudgetLocked = errors.New("odn: budget/milestone edit locked")
	// ErrMilestoneLocked 里程碑状态标记被拒(项目 ACCEPTED 后锁定):40900。
	ErrMilestoneLocked = errors.New("odn: milestone status locked")
)

// Milestone 施工项目里程碑(名称/计划完成日/状态;W6 000218)。
type Milestone struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	Name        string `json:"name"`
	PlannedDate string `json:"plannedDate,omitempty"` // YYYY-MM-DD,可空
	Status      string `json:"status"`
	DoneAt      string `json:"doneAt,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// ValidateMilestoneStatus 里程碑状态取值校验。
func ValidateMilestoneStatus(status string) error {
	if status == MPending || status == MDone {
		return nil
	}
	return ErrInvalidInput
}

// ValidateMilestoneEdit 预算/里程碑清单编辑窗口:仅项目 PENDING(BUILDING 前)可改。
func ValidateMilestoneEdit(projectStatus string) error {
	if projectStatus != CPending {
		return ErrBudgetLocked
	}
	return nil
}

// ValidateMilestoneMark 里程碑状态标记窗口:项目 ACCEPTED 后锁定(PENDING/BUILDING 可标记)。
func ValidateMilestoneMark(projectStatus string) error {
	if projectStatus == CAccepted {
		return ErrMilestoneLocked
	}
	return nil
}

// BudgetProgress 预算执行进度(只读派生:已结算=SETTLED 结算单合计,禁止直写)。
// 预算未登记时 BudgetAmount 为 nil,页面显示未登记而非 0(口径同 W2 未登记)。
type BudgetProgress struct {
	BudgetAmount  *float64 `json:"budgetAmount"`
	SettledAmount float64  `json:"settledAmount"`
}

// BudgetStore 预算与里程碑存储口(PGStore 实现;接口收敛在 ODNService)。
type BudgetStore interface {
	SetProjectBudget(ctx context.Context, projectID int64, amount *float64) error
	ListMilestones(ctx context.Context, projectID int64) ([]Milestone, error)
	AddMilestone(ctx context.Context, projectID int64, name, plannedDate string) (*Milestone, error)
	UpdateMilestone(ctx context.Context, milestoneID int64, name, plannedDate string) error
	MarkMilestone(ctx context.Context, milestoneID int64, status string) error
}
