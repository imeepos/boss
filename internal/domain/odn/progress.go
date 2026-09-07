package odn

import (
	"errors"
	"fmt"
)

// ProgressEntry 施工进度上报(P0-B,迁移 000213):资源级现场完成事实。
// 幂等键 (project_id, facility_code, client_msg_id):弱网重传不重复计量。
type ProgressEntry struct {
	ID           int64   `json:"id"`
	ProjectID    int64   `json:"projectId"`
	FacilityCode string  `json:"facilityCode"`
	DoneQty      float64 `json:"doneQty"`
	Lat          float64 `json:"lat,omitempty"`
	Lng          float64 `json:"lng,omitempty"`
	Note         string  `json:"note"`
	PhotoIDs     []int64 `json:"photoIds"`
	ClientMsgID  string  `json:"clientMsgId"`
	ReportedBy   int64   `json:"reportedBy"`
	ReportedAt   string  `json:"reportedAt,omitempty"`
}

// ErrFacilityNotInScope 上报设施不在项目施工范围内(明细未挂该设施)。
var ErrFacilityNotInScope = errors.New("odn: facility not in project scope")

// ValidateProgressEntry 进度上报纯校验:数量非负、坐标范围、消息 ID 与备注长度。
func ValidateProgressEntry(p ProgressEntry) error {
	if p.FacilityCode == "" {
		return fmt.Errorf("odn: progress facilityCode empty: %w", ErrInvalidInput)
	}
	if p.DoneQty < 0 {
		return fmt.Errorf("odn: progress doneQty=%v: %w", p.DoneQty, ErrInvalidInput)
	}
	if len(p.ClientMsgID) < 8 || len(p.ClientMsgID) > 64 {
		return fmt.Errorf("odn: progress clientMsgId len=%d: %w", len(p.ClientMsgID), ErrInvalidInput)
	}
	if len(p.Note) > 500 {
		return fmt.Errorf("odn: progress note len=%d: %w", len(p.Note), ErrInvalidInput)
	}
	if p.Lat != 0 && (p.Lat < -90 || p.Lat > 90) {
		return fmt.Errorf("odn: progress lat=%v: %w", p.Lat, ErrInvalidInput)
	}
	if p.Lng != 0 && (p.Lng < -180 || p.Lng > 180) {
		return fmt.Errorf("odn: progress lng=%v: %w", p.Lng, ErrInvalidInput)
	}
	return nil
}

// ItemProgress 清单项进度聚合:计划量(清单 quantity)与累计上报完成量。
type ItemProgress struct {
	FacilityCode string  `json:"facilityCode"`
	PlannedQty   float64 `json:"plannedQty"`
	DoneQty      float64 `json:"doneQty"`
	Entries      int64   `json:"entries"`
}
