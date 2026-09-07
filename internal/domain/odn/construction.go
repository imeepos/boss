package odn

import (
	"context"
	"errors"
)

// 施工项目状态(P6,T9,迁移 000199)。
const (
	CPending  = "PENDING"
	CBuilding = "BUILDING"
	CAccepted = "ACCEPTED"
)

// ErrInvalidProjStatus 非法施工单状态转移(未知/回退/终态再动)。
var ErrInvalidProjStatus = errors.New("odn: invalid project status transition")

// ErrItemLocked 明细变更被拒(ACCEPTED 后锁定/设施不可施工)。
var ErrItemLocked = errors.New("odn: construction item locked")

// ErrEmptyScope 开工前置缺失:施工范围为空(单内无任何资源明细),禁止空转开工。
var ErrEmptyScope = errors.New("odn: construction start blocked: empty resource scope")

// Construction 施工项目(含 as-built 竣工信息与承包商;000203/000204 扩展;W6 000218 预算执行)。
type Construction struct {
	ID             int64    `json:"id"`
	ProjNo         string   `json:"projNo"`
	Name           string   `json:"name"`
	PrvCode        string   `json:"prvCode,omitempty"`
	CityPrefix     string   `json:"cityPrefix,omitempty"`
	Status         string   `json:"status"`
	AsbuiltNote    string   `json:"asbuiltNote"`
	AcceptedBy     int64    `json:"acceptedBy"`
	AcceptedAt     string   `json:"acceptedAt,omitempty"`
	ItemCount      int64    `json:"itemCount"`
	UpdatedAt      string   `json:"updatedAt"`
	ContractorID   int64    `json:"contractorId"`   // 软引用 procurement_suppliers,0=未指定(存量兼容)
	ContractorName string   `json:"contractorName"` // 指定时名称快照
	ItemsAmount    float64  `json:"itemsAmount"`    // 清单金额汇总 SUM(amount),结算应付口径
	BudgetAmount   *float64 `json:"budgetAmount"`   // 预算金额,NULL=未登记(000218)
	SettledAmount  float64  `json:"settledAmount"`  // 已结算金额=SETTLED 结算单合计,只读派生(000218)
}

// ConstructionItem 单-设施明细(工程量清单行;amount 为生成列,后端计算)。
type ConstructionItem struct {
	ID           int64   `json:"id"`
	ProjectID    int64   `json:"projectId"`
	FacilityCode string  `json:"facilityCode"`
	Quantity     float64 `json:"quantity"`
	UnitPrice    float64 `json:"unitPrice"`
	Amount       float64 `json:"amount"`
}

// ValidateProjectTransition 施工单状态转移:线性 PENDING→BUILDING→ACCEPTED,同态 no-op。
func ValidateProjectTransition(from, to string) error {
	if from == to {
		return nil
	}
	switch {
	case from == CPending && to == CBuilding:
		return nil
	case from == CBuilding && to == CAccepted:
		return nil
	default:
		return ErrInvalidProjStatus
	}
}

// ValidateStartReady 开工前置纯校验:资源范围非空方可开工;状态合法性由 CAS 兜底。
func ValidateStartReady(itemCount int64) error {
	if itemCount <= 0 {
		return ErrEmptyScope
	}
	return nil
}

// ConstructionStore 施工项目存储口(PGStore 实现)。
type ConstructionStore interface {
	CreateProject(ctx context.Context, p Construction) error
	GetProject(ctx context.Context, id int64) (*Construction, error)
	ListProjects(ctx context.Context, limit int) ([]Construction, error)
	AddProjectItem(ctx context.Context, projectID int64, facilityCode string, qty, unitPrice float64) error
	UpdateProjectItem(ctx context.Context, projectID, itemID int64, qty, unitPrice float64) error
	StartProject(ctx context.Context, id int64) (int64, error)
	// AcceptProject 返回 (设施翻转数, 覆盖联动数, 错误);F6 竣工覆盖联动随事务(000211)。
	AcceptProject(ctx context.Context, id int64, acceptedBy int64, note string) (int64, int64, error)
	ListProjectItems(ctx context.Context, id int64) ([]ConstructionItem, error)
	// SetProjectContractor 指定/更换承包商(名称快照);存在有效结算单后锁定。
	SetProjectContractor(ctx context.Context, projectID, contractorID int64, contractorName string) error
}
