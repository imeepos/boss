package aaa

// AAA-A5(G8) 带宽-VSA 映射回归:模板解析/厂商属性对/配置覆盖/兜底路径。

import (
	"testing"
)

func TestParseBandwidthKbps(t *testing.T) {
	cases := []struct {
		in      string
		want    uint32
		wantOK  bool
	}{
		{in: "100M", want: 100000, wantOK: true},
		{in: "1000M", want: 1000000, wantOK: true},
		{in: " 50k ", want: 50, wantOK: true},
		{in: "1G", want: 1000000, wantOK: true},
		{in: "0M", wantOK: false},
		{in: "abc", wantOK: false},
		{in: "1.5M", wantOK: false},
		{in: "", wantOK: false},
	}
	for _, c := range cases {
		got, ok := ParseBandwidthKbps(c.in)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("ParseBandwidthKbps(%q)=(%d,%v) want (%d,%v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestBuildVSASpecDefaultsAndOverride(t *testing.T) {
	spec, err := BuildVSASpec(VSAConfig{})
	if err != nil {
		t.Fatal(err)
	}
	attrs := spec.ResolveRateVSA(NasVendorHuawei, "100M")
	if len(attrs) != 2 || attrs[0].VendorID != 2011 || attrs[0].Type != 78 || attrs[1].Type != 80 {
		t.Fatalf("huawei 默认对: %+v", attrs)
	}
	override, err := BuildVSASpec(VSAConfig{ZTESpec: "input-peak-rate,output-peak-rate"})
	if err != nil {
		t.Fatal(err)
	}
	zAttrs := override.ResolveRateVSA(NasVendorZTE, "100M")
	if len(zAttrs) != 2 || zAttrs[0].VendorID != 3902 || zAttrs[0].Type != 83 || zAttrs[1].Type != 85 {
		t.Fatalf("zte 覆盖对: %+v", zAttrs)
	}
	if _, err := BuildVSASpec(VSAConfig{HuaweiSpec: "no-such-attr,input-average-rate"}); err == nil {
		t.Fatal("未知属性名应启动期报错")
	}
	if _, err := BuildVSASpec(VSAConfig{ZTESpec: "only-one"}); err == nil {
		t.Fatal("格式错误应启动期报错")
	}
}

// 兜底路径:厂商不匹配/带宽不可解析/nil spec 一律返回空(调用方走字符串属性兜底)。
func TestResolveRateVSAFallbacks(t *testing.T) {
	spec, _ := BuildVSASpec(VSAConfig{})
	for _, c := range []struct {
		name      string
		vendor    NasVendor
		bandwidth string
	}{
		{name: "GENERIC 厂商", vendor: NasVendorGeneric, bandwidth: "100M"},
		{name: "未知厂商", vendor: NasVendor("MIKROTIK"), bandwidth: "100M"},
		{name: "空厂商", vendor: "", bandwidth: "100M"},
		{name: "带宽不可解析", vendor: NasVendorHuawei, bandwidth: "B-100"},
		{name: "带宽为空", vendor: NasVendorHuawei, bandwidth: ""},
	} {
		if got := spec.ResolveRateVSA(c.vendor, c.bandwidth); len(got) != 0 {
			t.Errorf("%s: got %+v want 空(兜底)", c.name, got)
		}
	}
	var nilSpec *VSASpec
	if got := nilSpec.ResolveRateVSA(NasVendorHuawei, "100M"); len(got) != 0 {
		t.Errorf("nil spec 应兜底: %+v", got)
	}
}