package odn

import "testing"

// TestValidateDeviceCode 资产编码规范 5.1 正则 + 3.1 同址扩容禁 -1。
func TestValidateDeviceCode(t *testing.T) {
	cases := []struct {
		code     string
		wantKind string
		wantErr  bool
	}{
		{"SNW001", DevSNW, false},
		{"OLT001", DevOLT, false},
		{"ODF001", DevODF, false},
		{"OCC001", DevOCC, false},
		{"ODB001", DevODB, false},
		{"SDB001", DevSDB, false},
		{"PRT001", DevPRT, false},
		{"TBP001", DevTBP, false},
		{"ODB001-2", DevODB, false},  // 同址扩容第 2 台
		{"ODB001-10", DevODB, false}, // 二位数扩容
		{"ODB001-1", "", true},       // 禁 -1(规范 3.1)
		{"OLT01", "", true},          // 位数不足
		{"OLT0001", "", true},
		{"XX001", "", true},
		{"olt001", "", true},
		{"ODB001-", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		kind, err := ValidateDeviceCode(c.code)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: 期望非法,实际通过 kind=%s", c.code, kind)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 意外错误 %v", c.code, err)
			continue
		}
		if kind != c.wantKind {
			t.Errorf("%s: kind=%s,期望 %s", c.code, kind, c.wantKind)
		}
	}
}

// TestRequiredParentKind 规范 2.1 归属链:ODB→OCC/SDB→ODB/PRT→SDB/TBP→PRT,顶层空。
func TestRequiredParentKind(t *testing.T) {
	cases := map[string]string{
		DevSNW: "", DevOLT: "", DevODF: "", DevOCC: "",
		DevODB: DevOCC, DevSDB: DevODB, DevPRT: DevSDB, DevTBP: DevPRT,
	}
	for kind, want := range cases {
		if got := RequiredParentKind(kind); got != want {
			t.Errorf("%s: 期望上级 %q,实际 %q", kind, want, got)
		}
	}
}

// TestNodeCode 局点编码拼接:城市前缀 + 3 位序号。
func TestNodeCode(t *testing.T) {
	s := Site{CityPrefix: "MNL", SiteNo: 1}
	if s.NodeCode() != "MNL001" {
		t.Fatalf("NodeCode=%s,期望 MNL001", s.NodeCode())
	}
	s.SiteNo = 42
	if s.NodeCode() != "MNL042" {
		t.Fatalf("NodeCode=%s,期望 MNL042", s.NodeCode())
	}
}
