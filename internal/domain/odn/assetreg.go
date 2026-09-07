package odn

import (
	"context"
	"errors"
	"regexp"
)

// ODN 资产化转固(P-INFRA-1 W8,迁移 000215;F7 资产转固阻塞闭环)。
// 桥表软引用裁定(adopted 2026-09-07-odn-asset-capitalization):
// odn×asset 关联落桥表 odn_asset_registrations,行=资产凭证(登记号/价值/来源/项目溯源),
// 不加跨域 FK;冲销 REVERSED 留历史不删(同 construction_settlements VOIDED 先例)。
const (
	RegFacility = "FACILITY" // 转固对象:无源设施(P/MH/TW/CLS/TBX)
	RegDevice   = "DEVICE"   // 转固对象:核心链路设备与 W3 导入域箱体(OCC/ODB/OBD/SDB/SBD 等)

	RegSourceProcurement  = "PROCUREMENT"  // 采购入库资产经项目出库转固
	RegSourceConstruction = "CONSTRUCTION" // 施工建成(须挂 ACCEPTED 项目)
	RegSourceDirect       = "DIRECT"       // 直购直转(无项目环节)

	RegActive   = "ACTIVE"   // 凭证有效:资产已转固(DEPLOYED)
	RegReversed = "REVERSED" // 已冲销:终态留历史
)

var regNoPattern = regexp.MustCompile(`^ZG-[0-9]{8}-[0-9]{5}$`)

var (
	// ErrRegConflict 对象/资产已有 ACTIVE 凭证、对象 RETIRED、资产状态不允许:40900。
	ErrRegConflict = errors.New("odn: asset registration conflict")
	// ErrRegState 凭证状态转移非法(冲销非 ACTIVE 等):40900。
	ErrRegState = errors.New("odn: invalid registration status")
	// ErrRegInput 登记入参不合法(价值<0/来源与项目不匹配/实体双列冲突):42200。
	ErrRegInput = errors.New("odn: invalid registration input")
)

// AssetRegistration 资产化凭证(桥表行):odn 对象 ↔ assets 的转固事实。
type AssetRegistration struct {
	ID                    int64   `json:"id"`
	RegistrationNo        string  `json:"registrationNo"`
	EntityKind            string  `json:"entityKind"`
	FacilityCode          string  `json:"facilityCode"`
	DeviceID              int64   `json:"deviceId"`
	AssetID               int64   `json:"assetId"`
	SourceKind            string  `json:"sourceKind"`
	ConstructionProjectID int64   `json:"constructionProjectId"`
	BatchID               int64   `json:"batchId"`
	ValueAmount           float64 `json:"valueAmount"`
	Status                string  `json:"status"`
	ReverseReason         string  `json:"reverseReason"`
	Remark                string  `json:"remark"`
	RegisteredBy          int64   `json:"registeredBy"`
	RegisteredAt          string  `json:"registeredAt"`
	ReversedBy            int64   `json:"reversedBy"`
	ReversedAt            string  `json:"reversedAt,omitempty"`
}

// EntityAssetReg 对象上的 ACTIVE 凭证摘要(设施/设备列表附加,资产身份视图可见)。
type EntityAssetReg struct {
	RegistrationNo string `json:"registrationNo"`
	AssetID        int64  `json:"assetId"`
	AssetCode      string `json:"assetCode"`
	AssetStatus    string `json:"assetStatus"`
}

// ValidateRegistrationInput 登记入参校验(42200 族):实体二选一、来源与项目匹配、价值非负。
func ValidateRegistrationInput(entityKind, sourceKind string, facilityCode string, deviceID, projectID int64, value float64) error {
	if entityKind != RegFacility && entityKind != RegDevice {
		return ErrRegInput
	}
	if entityKind == RegFacility && (facilityCode == "" || deviceID != 0) {
		return ErrRegInput
	}
	if entityKind == RegDevice && (deviceID <= 0 || facilityCode != "") {
		return ErrRegInput
	}
	if value < 0 {
		return ErrRegInput
	}
	switch sourceKind {
	case RegSourceProcurement, RegSourceDirect:
		if projectID != 0 {
			return ErrRegInput
		}
	case RegSourceConstruction:
		if projectID <= 0 {
			return ErrRegInput
		}
	default:
		return ErrRegInput
	}
	return nil
}

// ValidRegistrationNo 凭证号格式(导入/对账兜底校验)。
func ValidRegistrationNo(no string) bool { return regNoPattern.MatchString(no) }

// 资产化存储口(PGStore 实现;接口收敛在 ODNService)。
type AssetRegistrationStore interface {
	CreateRegistration(ctx context.Context, entityKind, facilityCode string, deviceID, assetID int64,
		sourceKind string, projectID int64, value float64, remark string, accountID int64) (*AssetRegistration, error)
	ListRegistrations(ctx context.Context, entityKind, entityRef, status string, limit int) ([]AssetRegistration, error)
	GetRegistration(ctx context.Context, id int64) (*AssetRegistration, error)
	ReverseRegistration(ctx context.Context, id, accountID int64, reason string) error
}
