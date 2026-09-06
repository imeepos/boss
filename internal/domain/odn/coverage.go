package odn

import (
	"context"
	"errors"
	"fmt"
	"math"
)

// 覆盖关联可装状态(fields.md 1.5.7,迁移 000197)。
const (
	CovServed   = "SERVED"   // 已覆盖可装机
	CovPending  = "PENDING"  // 规划在建(已挂规划目标)
	CovUnserved = "UNSERVED" // 未覆盖
)

// ResolveRadiusM 就近命中半径(米):超出判 UNSERVED。
// P1 用平面近似:菲律宾低纬度,网格级判定足够;精致化随 P2 端口占用再评估。
const ResolveRadiusM = 2000

// 覆盖关联领域错误。
var (
	// ErrInvalidCoverage 参数非法:状态越界或非 UNSERVED 未挂任何目标。
	ErrInvalidCoverage = errors.New("odn: invalid coverage")
	// ErrCoverageTarget 覆盖目标不存在(设施码/设备 id,FK 23503 归一)。
	ErrCoverageTarget = errors.New("odn: coverage target missing")
	// ErrNotServable 地址不可装机(覆盖门控拒单:未登记/未 SERVED;下单硬校验 T12)。
	ErrNotServable = errors.New("odn: address not servable")
)

// Coverage 地址覆盖关联(odn↔业务首桥;决策 adopted/2026-09-06-odn-business-linkage)。
type Coverage struct {
	ID           int64  `json:"id"`
	AddressID    int64  `json:"addressId"`
	FacilityCode string `json:"facilityCode"` // 服务设施,空=未挂
	DeviceID     int64  `json:"deviceId"`     // 服务核心设备,0=未挂
	Status       string `json:"status"`       // SERVED/PENDING/UNSERVED
	Note         string `json:"note"`
	AddressName  string `json:"addressName,omitempty"` // 列表联查 addresses.name
	UpdatedAt    string `json:"updatedAt"`
}

// CoverageResolved 可装性判定结果(ResolveLatLng 输出)。
type CoverageResolved struct {
	Status       string  `json:"status"` // SERVED/UNSERVED
	FacilityCode string  `json:"facilityCode,omitempty"`
	FacilityName string  `json:"facilityName,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lng          float64 `json:"lng,omitempty"`
	DistanceM    float64 `json:"distanceM,omitempty"`
}

// CoverageStore 覆盖关联存储口(PGStore 实现)。
type CoverageStore interface {
	UpsertCoverage(ctx context.Context, c Coverage) error
	GetCoverageByAddress(ctx context.Context, addressID int64) (*Coverage, error)
	ListCoverage(ctx context.Context, limit int) ([]Coverage, error)
	ResolveLatLng(ctx context.Context, lat, lng float64) (*CoverageResolved, error)
}

// ValidateCoverage 状态/目标一致性:非 UNSERVED 至少挂一个目标
// (域内先行拒绝,库端 address_coverage_target_chk 兜底)。
func ValidateCoverage(status, facilityCode string, deviceID int64) error {
	switch status {
	case CovServed, CovPending:
		if facilityCode == "" && deviceID <= 0 {
			return ErrInvalidCoverage
		}
		return nil
	case CovUnserved:
		return nil
	default:
		return ErrInvalidCoverage
	}
}

// haversineM 球面距离(米)。仅用于展示与半径阈值;SQL 排序用平面平方距离。
func haversineM(lat1, lng1, lat2, lng2 float64) float64 {
	const earthM = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthM * math.Asin(math.Sqrt(a))
}

// ResolveStatus 依据就近距离给出建议状态(<=2km 可装)。
func ResolveStatus(distanceM float64) string {
	if distanceM <= ResolveRadiusM {
		return CovServed
	}
	return CovUnserved
}

// CheckOrderCoverage 下单覆盖门控(P2,T12):地址须已登记且 SERVED 才放行。
// 无记录/未 SERVED → ErrNotServable(orders 发单号前拒单);由 order.Submit 调用。
func (s *PGStore) CheckOrderCoverage(ctx context.Context, addressID int64) error {
	cov, err := s.GetCoverageByAddress(ctx, addressID)
	if err != nil {
		return err
	}
	if cov == nil {
		return fmt.Errorf("%w: address=%d 未登记覆盖", ErrNotServable, addressID)
	}
	if cov.Status != CovServed {
		return fmt.Errorf("%w: address=%d status=%s", ErrNotServable, addressID, cov.Status)
	}
	return nil
}
