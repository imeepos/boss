package radius

// Access-Accept 厂商限速 VSA 组装:RFC 2865 §5.26 通用 Vendor-Specific 编码
// (Vendor-Id + Vendor-Type + Vendor-Length + integer),不引入厂商字典依赖。
// 编码失败输出 [aaa] 可 grep 告警(丢弃该属性,不阻断 Accept)。

import (
	"encoding/binary"
	"log"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// applyRateVSA 将厂商限速属性以通用 VSA 编码写入响应包。
func applyRateVSA(p *radius.Packet, attrs []aaa.VSAAttr) {
	for _, a := range attrs {
		val := make([]byte, 6) // Vendor-Type(1) + Vendor-Length(1) + integer(4)
		val[0] = a.Type
		val[1] = byte(len(val))
		binary.BigEndian.PutUint32(val[2:], a.Kbps)
		attr, err := radius.NewVendorSpecific(a.VendorID, val)
		if err != nil {
			log.Printf("[aaa] VSA ENCODE FAILED vendor=%d type=%d kbps=%d: %v",
				a.VendorID, a.Type, a.Kbps, err)
			continue
		}
		p.Add(rfc2865.VendorSpecific_Type, attr)
	}
}
