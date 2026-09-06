package aaa

// AAA-A5(G8):带宽模板 → 厂商限速 VSA 映射。属性语义名 → 厂商属性类型码内置表,
// 每厂商实际使用哪对属性经配置指定(不写死);厂商不匹配/无映射一律回退现状字符串属性
// (FramedPool 承载带宽模板串,行为不回退)。默认类型码:HUAWEI VID 2011 上行 78/下行 80
// (平均速率),ZTE VID 3902 上行 84/下行 86(平均速率);ZTE 码以目标设备 RADIUS 私有
// 属性规范为准,可用 BOSS_AAA_VSA_ZTE 覆盖(fields.md §8J)。

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// VSAAttr 单个厂商限速属性(类型码 + kbps 值;radius 包负责 VSA 线编码)。
type VSAAttr struct {
	VendorID uint32
	Type     byte
	Name     string // 语义名(日志/契约可读)
	Kbps     uint32
}

// 厂商注册标识(IANA 分配,非配置项)与属性语义名 → 类型码表。
var (
	huaweiVendorID uint32 = 2011
	zteVendorID    uint32 = 3902
	// Huawei RADIUS 属性规范 1.1:78/79 上行平均/峰值,80/81 下行平均/峰值(kbps)。
	huaweiAttrCodes = map[string]byte{
		"input-average-rate":  78,
		"input-peak-rate":     79,
		"output-average-rate": 80,
		"output-peak-rate":    81,
	}
	// ZTE 私有属性(镜像华为布局偏移 5):83/84 上行峰值/平均,85/86 下行峰值/平均。
	zteAttrCodes = map[string]byte{
		"input-average-rate":  84,
		"input-peak-rate":     83,
		"output-average-rate": 86,
		"output-peak-rate":    85,
	}
)

// VSAConfig 厂商限速属性配置:每厂商 "上行属性名,下行属性名"(空=厂商默认对)。
type VSAConfig struct {
	HuaweiSpec string // BOSS_AAA_VSA_HUAWEI,默认 input-average-rate,output-average-rate
	ZTESpec    string // BOSS_AAA_VSA_ZTE
}

// VSASpec 启动期解析完成的厂商限速属性对(运行期只查表,不再解析配置)。
type VSASpec struct {
	huawei *vendorRateSpec
	zte    *vendorRateSpec
}

type vendorRateSpec struct {
	vendorID uint32
	upType   byte
	downType byte
	upName   string
	downName string
}

// BuildVSASpec 启动期解析配置;非法属性名直接报错(配置错误应在启动期暴露)。
func BuildVSASpec(cfg VSAConfig) (*VSASpec, error) {
	h, err := parseVendorSpec(huaweiVendorID, huaweiAttrCodes, cfg.HuaweiSpec,
		"input-average-rate", "output-average-rate")
	if err != nil {
		return nil, fmt.Errorf("aaa: vsa huawei: %w", err)
	}
	z, err := parseVendorSpec(zteVendorID, zteAttrCodes, cfg.ZTESpec,
		"input-average-rate", "output-average-rate")
	if err != nil {
		return nil, fmt.Errorf("aaa: vsa zte: %w", err)
	}
	return &VSASpec{huawei: h, zte: z}, nil
}

// parseVendorSpec 解析单厂商属性对;格式 "上行名,下行名",空=默认对。
func parseVendorSpec(vid uint32, codes map[string]byte, spec, defUp, defDown string) (*vendorRateSpec, error) {
	up, down := defUp, defDown
	if spec != "" {
		parts := strings.Split(spec, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("expect up,down got %q", spec)
		}
		up, down = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	for _, n := range []string{up, down} {
		if _, ok := codes[n]; !ok {
			return nil, fmt.Errorf("unknown attr name %q", n)
		}
	}
	return &vendorRateSpec{
		vendorID: vid, upType: codes[up], downType: codes[down], upName: up, downName: down,
	}, nil
}

// ResolveRateVSA 带宽模板 → 厂商限速属性;厂商未匹配或带宽不可解析返回 nil(调用方兜底)。
func (s *VSASpec) ResolveRateVSA(vendor NasVendor, bandwidth string) []VSAAttr {
	if s == nil {
		return nil
	}
	var sp *vendorRateSpec
	switch vendor {
	case NasVendorHuawei:
		sp = s.huawei
	case NasVendorZTE:
		sp = s.zte
	default:
		return nil
	}
	kbps, ok := ParseBandwidthKbps(bandwidth)
	if !ok {
		return nil
	}
	return []VSAAttr{
		{VendorID: sp.vendorID, Type: sp.upType, Name: sp.upName, Kbps: kbps},
		{VendorID: sp.vendorID, Type: sp.downType, Name: sp.downName, Kbps: kbps},
	}
}

// ParseBandwidthKbps 带宽模板(如 100M/500M/1000M/50K)→ kbps;不识别返回 false。
// 口径与 product_offers.bandwidth 注释一致(如 300M/500M/1000M,K/M/G 十进制)。
func ParseBandwidthKbps(bw string) (uint32, bool) {
	bw = strings.ToUpper(strings.TrimSpace(bw))
	var mult uint64
	switch {
	case strings.HasSuffix(bw, "G"):
		mult, bw = 1000*1000, strings.TrimSuffix(bw, "G")
	case strings.HasSuffix(bw, "M"):
		mult, bw = 1000, strings.TrimSuffix(bw, "M")
	case strings.HasSuffix(bw, "K"):
		mult, bw = 1, strings.TrimSuffix(bw, "K")
	default:
		return 0, false
	}
	n, err := strconv.ParseUint(bw, 10, 32)
	if err != nil || n == 0 {
		return 0, false
	}
	v := uint64(n) * mult
	if v > math.MaxUint32 {
		return 0, false
	}
	return uint32(v), true
}
