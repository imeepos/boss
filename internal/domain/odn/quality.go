package odn

import (
	"errors"
	"fmt"
)

// 资源级测试与整改(P0-C,迁移 000214):验收证据与闭环。

const (
	TestOTDR         = "OTDR"
	TestOpticalPower = "OPTICAL_POWER"
	TestConnectivity = "CONNECTIVITY"
	TestPass         = "PASS"
	TestFail         = "FAIL"
)

const (
	DefectOpen       = "OPEN"
	DefectRectifying = "RECTIFYING"
	DefectVerified   = "VERIFIED"
)

// ErrOpenDefects 竣工前置缺失:同项目存在未 VERIFIED 整改项,禁止验收。
var ErrOpenDefects = errors.New("odn: accept blocked by unverified defects")

// ErrDefectState 整改项状态转移被拒(终态再动/非法转移)。
var ErrDefectState = errors.New("odn: invalid defect state transition")

// QualityTest 资源级测试记录(append-only,每次读数即事实)。
type QualityTest struct {
	ID            int64   `json:"id"`
	ProjectID     int64   `json:"projectId"`
	ResourceType  string  `json:"resourceType"`
	ResourceRef   string  `json:"resourceRef"`
	TestKind      string  `json:"testKind"`
	Result        string  `json:"result"`
	AttenuationDB float64 `json:"attenuationDb"`
	PowerDBM      float64 `json:"powerDbm"`
	Note          string  `json:"note"`
	ReportedBy    int64   `json:"reportedBy"`
	ReportedAt    string  `json:"reportedAt,omitempty"`
}

// QualityDefect 整改项:OPEN→RECTIFYING→VERIFIED(OPEN 可直达 VERIFIED,现场即改即验)。
type QualityDefect struct {
	ID           int64  `json:"id"`
	ProjectID    int64  `json:"projectId"`
	FacilityCode string `json:"facilityCode"`
	Severity     string `json:"severity"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	Note         string `json:"note"`
	OpenedBy     int64  `json:"openedBy"`
	OpenedAt     string `json:"openedAt,omitempty"`
	RectifiedAt  string `json:"rectifiedAt,omitempty"`
	VerifiedBy   int64  `json:"verifiedBy"`
	VerifiedAt   string `json:"verifiedAt,omitempty"`
}

var validTestKinds = map[string]bool{TestOTDR: true, TestOpticalPower: true, TestConnectivity: true}

var validResourceTypes = map[string]bool{"FACILITY": true, "SEGMENT": true, "FIBER": true, "PORT": true}

// ValidateQualityTest 测试记录纯校验:资源类型/测试类型/结果/指标范围。
func ValidateQualityTest(t QualityTest) error {
	if !validResourceTypes[t.ResourceType] {
		return fmt.Errorf("odn: test resourceType=%s: %w", t.ResourceType, ErrInvalidInput)
	}
	if t.ResourceRef == "" {
		return fmt.Errorf("odn: test resourceRef empty: %w", ErrInvalidInput)
	}
	if !validTestKinds[t.TestKind] {
		return fmt.Errorf("odn: test kind=%s: %w", t.TestKind, ErrInvalidInput)
	}
	if t.Result != TestPass && t.Result != TestFail {
		return fmt.Errorf("odn: test result=%s: %w", t.Result, ErrInvalidInput)
	}
	if t.AttenuationDB < 0 {
		return fmt.Errorf("odn: test attenuation negative: %w", ErrInvalidInput)
	}
	// power_dbm 为 dBm,典型值为负;0 视为未录,不作符号限制。
	return nil
}

// ValidateDefectOpen 开单校验:严重度枚举 + 描述非空。
func ValidateDefectOpen(d QualityDefect) error {
	switch d.Severity {
	case "MINOR", "MAJOR", "CRITICAL":
	default:
		return fmt.Errorf("odn: defect severity=%s: %w", d.Severity, ErrInvalidInput)
	}
	if d.Description == "" {
		return fmt.Errorf("odn: defect description empty: %w", ErrInvalidInput)
	}
	return nil
}

// ValidateDefectTransition 整改状态机:OPEN/RECTIFYING→RECTIFYING(整改),OPEN/RECTIFYING→VERIFIED(复验),终态再动拒绝。
func ValidateDefectTransition(from, to string) error {
	if from == DefectVerified {
		return ErrDefectState
	}
	switch {
	case to == DefectRectifying && (from == DefectOpen || from == DefectRectifying):
		return nil
	case to == DefectVerified && (from == DefectOpen || from == DefectRectifying):
		return nil
	default:
		return ErrDefectState
	}
}
