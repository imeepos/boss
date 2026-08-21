package odn

import (
	"errors"
	"regexp"
)

// 光缆段落错误。
var (
	// ErrInvalidEndpoint 端点编码非法或不支持作为段落端点。
	ErrInvalidEndpoint = errors.New("odn: invalid segment endpoint")
	// ErrSamePriority 两端设备同优先级,无法定向(规范 5.2 只定义了单向序)。
	ErrSamePriority = errors.New("odn: endpoints same priority")
)

// endpointCode 段落端点:设施码(5 位)或核心设备码(3 位,未来 odn 设备实体)。
var endpointCode = regexp.MustCompile(`^(P|MH|TW|CLS|TBX|SNW|ODF|OCC|ODB|SDB|PRT|TBP|OLT)[0-9]{3,5}$`)

// endpointPriority 端点设备优先级(规范 5.2,数字越小优先级越高;A 端=较小值)。
// TW/TBX 规范未列,作最低档兜底(高于仅同档互连禁止,注释留痕)。
var endpointPriority = map[string]int{
	"SNW": 1, "ODF": 2, "OCC": 3, "CLS": 4, "ODB": 5,
	"P": 6, "MH": 7, "TW": 8, "TBX": 8, "OLT": 8, "SDB": 8, "PRT": 8, "TBP": 8,
}

// Segment 光缆段落(A/B 端已按优先级定向)。
type Segment struct {
	ID    int64  `json:"id"`
	ACode string `json:"aCode"` // A 端(优先级较高设备)
	BCode string `json:"bCode"` // B 端(优先级较低设备)
	Name  string `json:"name"`
}

// Fiber 段落内光缆(G01~G99)。
type Fiber struct {
	SegmentID int64  `json:"segmentId"`
	GNo       int16  `json:"gNo"` // 01~99
	Kind      string `json:"kind"`
}

// OrderEndpoints 按规范 5.2 定向:返回 (A端, B端);同优先级报错。
func OrderEndpoints(code1, code2 string) (a, b string, err error) {
	k1, ok1 := endpointKind(code1)
	k2, ok2 := endpointKind(code2)
	if !ok1 || !ok2 {
		return "", "", ErrInvalidEndpoint
	}
	p1, p2 := endpointPriority[k1], endpointPriority[k2]
	if p1 == p2 {
		return "", "", ErrSamePriority
	}
	if p1 < p2 {
		return code1, code2, nil
	}
	return code2, code1, nil
}

// endpointKind 提取端点编码的设备类型前缀。
func endpointKind(code string) (string, bool) {
	m := endpointCode.FindStringSubmatch(code)
	if m == nil {
		return "", false
	}
	return m[1], true
}
