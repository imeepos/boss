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

// Construction 施工项目(含 as-built 竣工信息)。
type Construction struct {
	ID          int64  `json:"id"`
	ProjNo      string `json:"projNo"`
	Name        string `json:"name"`
	PrvCode     string `json:"prvCode,omitempty"`
	CityPrefix  string `json:"cityPrefix,omitempty"`
	Status      string `json:"status"`
	AsbuiltNote string `json:"asbuiltNote"`
	AcceptedBy  int64  `json:"acceptedBy"`
	AcceptedAt  string `json:"acceptedAt,omitempty"`
	ItemCount   int64  `json:"itemCount"`
	UpdatedAt   string `json:"updatedAt"`
}

// ConstructionItem 单-设施明细。
type ConstructionItem struct {
	ID           int64  `json:"id"`
	ProjectID    int64  `json:"projectId"`
	FacilityCode string `json:"facilityCode"`
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

// ConstructionStore 施工项目存储口(PGStore 实现)。
type ConstructionStore interface {
	CreateProject(ctx context.Context, p Construction) error
	GetProject(ctx context.Context, id int64) (*Construction, error)
	ListProjects(ctx context.Context, limit int) ([]Construction, error)
	AddProjectItem(ctx context.Context, projectID int64, facilityCode string) error
	StartProject(ctx context.Context, id int64) (int64, error)
	AcceptProject(ctx context.Context, id int64, acceptedBy int64, note string) (int64, error)
	ListProjectItems(ctx context.Context, id int64) ([]ConstructionItem, error)
}
