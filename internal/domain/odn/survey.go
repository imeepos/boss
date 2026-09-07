package odn

import (
	"errors"
	"fmt"
)

// 勘测任务域(P-INFRA-1 W7,F5b):admin 创建/指派,师傅端可见可接,现场回填留痕。
// 状态机: PENDING(待执行) -> ACCEPTED(师傅已接) -> BACKFILLED(已回填);
// PENDING/ACCEPTED -> CANCELLED(admin 取消)。BACKFILLED 后仍可追加回填(多勘测点,append-only)。

// 勘测任务状态(terms.md §4)。
const (
	SurveyPending    = "PENDING"
	SurveyAccepted   = "ACCEPTED"
	SurveyBackfilled = "BACKFILLED"
	SurveyCancelled  = "CANCELLED"
)

// 勘测建议(terms.md §4):可装 / 需新建设施。
const (
	SuggestCanInstall      = "CAN_INSTALL"
	SuggestNeedNewFacility = "NEED_NEW_FACILITY"
)

// ErrSurveyState 勘测任务状态不允许该操作(未接单即回填/终态再指派等)。
var ErrSurveyState = errors.New("odn: survey task state disallows operation")

// ErrSurveyNotAssignee 非该任务指派师傅(回填/接单越权)。
var ErrSurveyNotAssignee = errors.New("odn: survey task not assigned to caller")

// SurveyTask 勘测任务。
type SurveyTask struct {
	ID               int64  `json:"id"`
	TaskNo           string `json:"taskNo"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	PrvCode          string `json:"prvCode"`
	CityPrefix       string `json:"cityPrefix"`
	GridCode         int64  `json:"gridCode"`
	AssignedWorkerID int64  `json:"assignedWorkerId"`
	WorkerName       string `json:"workerName,omitempty"`
	Status           string `json:"status"`
	ReportCount      int64  `json:"reportCount"`
	CreatedBy        int64  `json:"createdBy"`
	CreatedAt        string `json:"createdAt,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
}

// SurveyReport 勘测回填(append-only:只增不改不删,重复 clientMsgId 幂等回显)。
type SurveyReport struct {
	ID           int64   `json:"id"`
	TaskID       int64   `json:"taskId"`
	WorkerID     int64   `json:"workerId"`
	WorkerName   string  `json:"workerName,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lng          float64 `json:"lng,omitempty"`
	FacilityNote string  `json:"facilityNote"`
	Suggestion   string  `json:"suggestion"`
	PhotoIDs     []int64 `json:"photoIds"`
	ClientMsgID  string  `json:"clientMsgId"`
	ReportedAt   string  `json:"reportedAt,omitempty"`
}

// SurveyCreateInput admin 创建勘测任务入参。
type SurveyCreateInput struct {
	Title            string
	Description      string
	PrvCode          string
	CityPrefix       string
	GridCode         int64
	AssignedWorkerID int64
	CreatedBy        int64
}

// ValidateSurveyCreate 纯校验:标题必填、说明/坐标边界、建议枚举外的字段走 DB 约束兜底。
func ValidateSurveyCreate(in SurveyCreateInput) error {
	if len(in.Title) == 0 || len(in.Title) > 128 {
		return fmt.Errorf("odn: survey title len=%d: %w", len(in.Title), ErrInvalidInput)
	}
	if len(in.Description) > 500 {
		return fmt.Errorf("odn: survey description len=%d: %w", len(in.Description), ErrInvalidInput)
	}
	return nil
}

// ValidateSurveyReport 回填纯校验:建议枚举、坐标范围、备注长度、幂等键长度。
func ValidateSurveyReport(r SurveyReport) error {
	if r.Suggestion != SuggestCanInstall && r.Suggestion != SuggestNeedNewFacility {
		return fmt.Errorf("odn: survey suggestion=%s: %w", r.Suggestion, ErrInvalidInput)
	}
	if r.WorkerID <= 0 {
		return fmt.Errorf("odn: survey workerId=%d: %w", r.WorkerID, ErrInvalidInput)
	}
	if r.Lat != 0 && (r.Lat < -90 || r.Lat > 90) {
		return fmt.Errorf("odn: survey lat=%v: %w", r.Lat, ErrInvalidInput)
	}
	if r.Lng != 0 && (r.Lng < -180 || r.Lng > 180) {
		return fmt.Errorf("odn: survey lng=%v: %w", r.Lng, ErrInvalidInput)
	}
	if len(r.FacilityNote) > 500 {
		return fmt.Errorf("odn: survey facilityNote len=%d: %w", len(r.FacilityNote), ErrInvalidInput)
	}
	if len(r.ClientMsgID) < 8 || len(r.ClientMsgID) > 64 {
		return fmt.Errorf("odn: survey clientMsgId len=%d: %w", len(r.ClientMsgID), ErrInvalidInput)
	}
	return nil
}
