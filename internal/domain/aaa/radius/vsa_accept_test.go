package radius

// AAA-A5(G8) Access-Accept 厂商 VSA 下发回归:华为/中兴各一 + 无映射兜底(行为不回退)。

import (
	"encoding/binary"
	"testing"

	"layeh.com/radius"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// vsaKey 响应包中一个 VSA 属性的定位(厂商 ID + 厂商属性类型)。
type vsaKey struct {
	vid  uint32
	vtype byte
}

// collectVSA 遍历响应包 AVP,收集 Vendor-Specific(type 26)属性为 {vid,vtype}:value。
func collectVSA(t *testing.T, p *radius.Packet) map[vsaKey]uint32 {
	t.Helper()
	out := map[vsaKey]uint32{}
	for _, attr := range p.Attributes {
		if attr.Type != 26 {
			continue
		}
		vid, val, err := radius.VendorSpecific(attr.Attribute)
		if err != nil || len(val) != 6 {
			continue
		}
		out[vsaKey{vid: vid, vtype: val[0]}] = binary.BigEndian.Uint32(val[2:6])
	}
	return out
}

// framedPool 响应包中 FramedPool(type 88)的字符串值;缺省为空。
func framedPool(p *radius.Packet) string {
	for _, attr := range p.Attributes {
		if attr.Type == 88 {
			return string(attr.Attribute)
		}
	}
	return ""
}

func acceptPacket(t *testing.T, vendor aaa.NasVendor) *radius.Packet {
	t.Helper()
	spec, err := aaa.BuildVSASpec(aaa.VSAConfig{})
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		Auth: &authStub{decision: aaa.Decision{Authorize: true, Bandwidth: "100M", SessionTTL: 60}},
		Nas:  nasResolverStub{nas: &aaa.NasAuth{Client: aaa.NasClient{Vendor: vendor}}},
		VSA:  spec,
	}
	w := &responseStub{}
	req := accessRequest("LOID-V")
	req.RemoteAddr = remoteAt("10.1.1.9")
	h.ServeRADIUS(w, req)
	if w.packet == nil || w.packet.Code != radius.CodeAccessAccept {
		t.Fatalf("accept failed: %v", w.packet)
	}
	return w.packet
}

func TestAcceptHuaweiVSA(t *testing.T) {
	attrs := collectVSA(t, acceptPacket(t, aaa.NasVendorHuawei))
	if attrs[vsaKey{vid: 2011, vtype: 78}] != 100000 || attrs[vsaKey{vid: 2011, vtype: 80}] != 100000 {
		t.Fatalf("华为 VSA(78 上行/80 下行,100M=100000kbps): %v", attrs)
	}
	if framedPool(acceptPacket(t, aaa.NasVendorHuawei)) != "" {
		t.Fatal("命中 VSA 时不应再带 FramedPool 带宽串")
	}
}

func TestAcceptZTEVSA(t *testing.T) {
	attrs := collectVSA(t, acceptPacket(t, aaa.NasVendorZTE))
	if attrs[vsaKey{vid: 3902, vtype: 84}] != 100000 || attrs[vsaKey{vid: 3902, vtype: 86}] != 100000 {
		t.Fatalf("中兴 VSA(84 上行/86 下行,100M=100000kbps): %v", attrs)
	}
}

func TestAcceptFallbackWithoutMapping(t *testing.T) {
	// GENERIC 厂商与无注册表两条兜底路径:保留现状 FramedPool 带宽串,不下发 VSA。
	for _, tc := range []struct {
		name   string
		vendor aaa.NasVendor
		nas    bool
	}{
		{name: "GENERIC 厂商兜底", vendor: aaa.NasVendorGeneric, nas: true},
		{name: "无注册表兜底", nas: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec, _ := aaa.BuildVSASpec(aaa.VSAConfig{})
			h := &Handler{
				Auth: &authStub{decision: aaa.Decision{Authorize: true, Bandwidth: "B-100", SessionTTL: 60}},
				VSA:  spec,
			}
			if tc.nas {
				h.Nas = nasResolverStub{nas: &aaa.NasAuth{Client: aaa.NasClient{Vendor: tc.vendor}}}
			}
			w := &responseStub{}
			req := accessRequest("LOID-F")
			req.RemoteAddr = remoteAt("10.1.1.9")
			h.ServeRADIUS(w, req)
			if len(collectVSA(t, w.packet)) != 0 {
				t.Fatal("兜底路径不应下发 VSA")
			}
			if framedPool(w.packet) != "B-100" {
				t.Fatalf("兜底应保留带宽串: %q", framedPool(w.packet))
			}
		})
	}
}