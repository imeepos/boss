package odn

import (
	"context"
	"errors"
)

// 材料出库单(P-INFRA-1 W8,迁移 000215;F7 材料资产出库后从台账消失)。
// 台账连续性:OPEN=备出库(资产仍 IN_STOCK);CONFIRMED=已出库至项目工地(资产 IN_TRANSIT,000216);
// CANCELLED=取消/退库(在途资产回 IN_STOCK)。成本归集归 W9 项目领料,本单只管台账事实。
const (
	IssueOpen      = "OPEN"
	IssueConfirmed = "CONFIRMED"
	IssueCancelled = "CANCELLED"
)

var (
	// ErrIssueState 出库单状态转移非法:40900。
	ErrIssueState = errors.New("odn: invalid material issue status")
	// ErrIssueAsset 资产不可出库(非 IN_STOCK/不存在/重复行):40900。
	ErrIssueAsset = errors.New("odn: asset not issuable")
	// ErrIssueInput 出库入参不合法(空明细等):42200。
	ErrIssueInput = errors.New("odn: invalid material issue input")
)

// MaterialIssue 材料出库单头(项目软引用 + 单号快照,口径同结算单 contractor 快照)。
type MaterialIssue struct {
	ID          int64   `json:"id"`
	IssueNo     string  `json:"issueNo"`
	ProjectID   int64   `json:"projectId"`
	ProjectNo   string  `json:"projectNo"`
	Status      string  `json:"status"`
	Remark      string  `json:"remark"`
	AssetIDs    []int64 `json:"assetIds"`
	CreatedBy   int64   `json:"createdBy"`
	IssuedBy    int64   `json:"issuedBy"`
	CancelledBy int64   `json:"cancelledBy"`
	CreatedAt   string  `json:"createdAt"`
	IssuedAt    string  `json:"issuedAt,omitempty"`
	CancelledAt string  `json:"cancelledAt,omitempty"`
}

// ValidateIssueTransition 出库单状态转移:OPEN→CONFIRMED/CANCELLED;CONFIRMED→CANCELLED(退库);
// CANCELLED 终态;同态 no-op。
func ValidateIssueTransition(from, to string) error {
	if from == to {
		return nil
	}
	switch {
	case from == IssueOpen && to == IssueConfirmed:
		return nil
	case from == IssueOpen && to == IssueCancelled:
		return nil
	case from == IssueConfirmed && to == IssueCancelled:
		return nil
	default:
		return ErrIssueState
	}
}

// 材料出库存储口(PGStore 实现;接口收敛在 ODNService)。
type MaterialIssueStore interface {
	CreateIssue(ctx context.Context, projectID int64, assetIDs []int64, remark string, accountID int64) (*MaterialIssue, error)
	ListIssues(ctx context.Context, projectID int64, status string, limit int) ([]MaterialIssue, error)
	GetIssue(ctx context.Context, id int64) (*MaterialIssue, error)
	ConfirmIssue(ctx context.Context, id, accountID int64) error
	CancelIssue(ctx context.Context, id, accountID int64) error
}
