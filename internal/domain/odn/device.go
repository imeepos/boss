package odn

import (
	"errors"
	"fmt"
	"regexp"
)

// 设备域错误。
var (
	// ErrInvalidDeviceCode 设备码格式非法(规范 2.2 / 5.1 正则)。
	ErrInvalidDeviceCode = errors.New("odn: invalid device code")
	// ErrBadHierarchy 归属链非法(规范 2.1:ODB→OCC/SDB→ODB/PRT→SDB/TBP→PRT)。
	ErrBadHierarchy = errors.New("odn: bad hierarchy")
)

// 核心链路设备类型(资产编码规范 2.2;OBD/SBD 为箱内部件扩展,规范 2.4 口径,
// 裁定见 docs/notes/adopted/2026-09-07-odn-box-types-import-chain.md)。
const (
	DevSNW = "SNW" // 机房根节点,全网唯一
	DevOLT = "OLT" // 市域唯一
	DevODF = "ODF" // 按需节点,市域唯一
	DevOCC = "OCC" // 按需节点,市域唯一
	DevODB = "ODB" // 归属 OCC
	DevOBD = "OBD" // 一级分光器,箱内部件(规范 2.4),归属 ODB
	DevSDB = "SDB" // 归属 ODB(二级分光必选)
	DevSBD = "SBD" // 二级分光器,箱内部件(扩展裁定),归属 SDB
	DevPRT = "PRT" // 归属 SDB
	DevTBP = "TBP" // 归属 PRT,可选末端
)

// BoxKinds 箱体设备类型(W3 导入域:允许无城市,编码规范 2.2/2.4)。
var BoxKinds = []string{DevODF, DevOCC, DevODB, DevOBD, DevSDB, DevSBD}

// parentKind 归属链(规范 2.1 拓扑 + 2.4 箱内部件;空=顶层,可挂局点)。
var parentKind = map[string]string{
	DevODB: DevOCC, DevOBD: DevODB, DevSDB: DevODB,
	DevSBD: DevSDB, DevPRT: DevSDB, DevTBP: DevPRT,
}

// deviceCode 设备码:3 字母前缀 + 3 位编号 + 可选同址扩容后缀 -N(N>=2,禁 -1,规范 3.1)。
var deviceCode = regexp.MustCompile(`^(SNW|OLT|ODF|OCC|ODB|OBD|SDB|SBD|PRT|TBP)([0-9]{3})(-([2-9]|[1-9][0-9]+))?$`)

// Site 局点(NodeCode = CityPrefix + 3 位序号)。
type Site struct {
	PrvCode         string  `json:"prvCode"`
	CityPrefix      string  `json:"cityPrefix"`
	SiteNo          int16   `json:"siteNo"` // 001~999
	Name            string  `json:"name"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	Status          string  `json:"status"`          // ACTIVE/RETIRED
	LifecycleStatus string  `json:"lifecycleStatus"` // PLANNED/IN_BUILD/IN_SERVICE/RETIRED(000198)
}

// NodeCode 局点编码,如 MNL001。
func (s Site) NodeCode() string {
	return fmt.Sprintf("%s%03d", s.CityPrefix, s.SiteNo)
}

// Device 核心链路设备。
type Device struct {
	ID              int64    `json:"id"`
	Code            string   `json:"code"` // OLT001/ODB001-2
	Kind            string   `json:"kind"`
	PrvCode         string   `json:"prvCode"`
	CityPrefix      string   `json:"cityPrefix"`
	SiteNo          int16    `json:"siteNo"`   // 0=市域设备不挂局点
	ParentID        int64    `json:"parentId"` // 0=顶层
	Name            string   `json:"name"`
	Lat             *float64 `json:"lat"` // 可空;无坐标设备不上地图点位
	Lng             *float64 `json:"lng"`
	Status          string   `json:"status"`          // IN_USE/RETIRED
	LifecycleStatus string   `json:"lifecycleStatus"` // PLANNED/IN_BUILD/IN_SERVICE/RETIRED(000198)
}

// ValidateDeviceCode 校验设备码格式并解析 kind(规范 5.1 正则 + 3.1 禁 -1)。
func ValidateDeviceCode(code string) (kind string, err error) {
	m := deviceCode.FindStringSubmatch(code)
	if m == nil {
		return "", ErrInvalidDeviceCode
	}
	return m[1], nil
}

// RequiredParentKind 设备的上级类型要求;空=顶层(SNW/OLT/ODF/OCC)。
func RequiredParentKind(kind string) string { return parentKind[kind] }

// SiteService / DeviceService 方法挂在 ODNService(odn.go)。
