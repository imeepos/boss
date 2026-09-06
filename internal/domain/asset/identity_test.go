// 身份三要素与 EPC 校验单测(P3-T2):归一口径、格式闸门、编辑指针语义。
package asset

import (
	"errors"
	"testing"
)

func TestNormalizeIdentity(t *testing.T) {
	cases := []struct {
		name       string
		sn, mac, l string
		wantSN     string
		wantMAC    string
		wantLOID   string
		wantErr    error
	}{
		{name: "全空归空", sn: "", mac: "", l: "", wantSN: "", wantMAC: "", wantLOID: ""},
		{name: "空白串归空", sn: "  ", mac: " ", l: "\t", wantSN: "", wantMAC: "", wantLOID: ""},
		{name: "冒号MAC原样入库", sn: " SN001 ", mac: "AA:BB:CC:DD:EE:01", l: " L001 ",
			wantSN: "SN001", wantMAC: "AA:BB:CC:DD:EE:01", wantLOID: "L001"},
		{name: "横杠MAC合法", sn: "S", mac: "AA-BB-CC-DD-EE-01", l: "L",
			wantSN: "S", wantMAC: "AA-BB-CC-DD-EE-01", wantLOID: "L"},
		{name: "小写hexMAC合法", sn: "", mac: "aa:bb:cc:dd:ee:01", l: "",
			wantSN: "", wantMAC: "aa:bb:cc:dd:ee:01", wantLOID: ""},
		{name: "位数不足拒绝", sn: "", mac: "AA:BB:CC:DD:EE", l: "", wantErr: ErrInvalidMAC},
		{name: "非hex拒绝", sn: "", mac: "AA:BB:CC:DD:EE:ZZ", l: "", wantErr: ErrInvalidMAC},
		{name: "分隔符混用拒绝", sn: "", mac: "AA-BB:CC-DD:EE:01", l: "", wantErr: ErrInvalidMAC},
		{name: "无分隔符拒绝", sn: "", mac: "AABBCCDDEE01", l: "", wantErr: ErrInvalidMAC},
	}
	for _, tc := range cases {
		sn, mac, loid, err := NormalizeIdentity(tc.sn, tc.mac, tc.l)
		if tc.wantErr != nil {
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("%s: err = %v, want %v", tc.name, err, tc.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected err %v", tc.name, err)
			continue
		}
		if sn != tc.wantSN || mac != tc.wantMAC || loid != tc.wantLOID {
			t.Errorf("%s: got (%q,%q,%q), want (%q,%q,%q)",
				tc.name, sn, mac, loid, tc.wantSN, tc.wantMAC, tc.wantLOID)
		}
	}
}

func TestNormalizeEPC(t *testing.T) {
	// 合法:SGTIN-96 头部,小写归大写。
	v, err := NormalizeEPC(" 30a1b2c3d4e5f60718293a4b ")
	if err != nil || v != "30A1B2C3D4E5F60718293A4B" {
		t.Errorf("valid lowercase: v=%q err=%v", v, err)
	}
	for _, head := range []string{"32", "33", "35"} {
		epc := head + "A1B2C3D4E5F60718293A4B"
		if _, err := NormalizeEPC(epc); err != nil {
			t.Errorf("head %s: unexpected err %v", head, err)
		}
	}
	// 位数不足。
	if _, err := NormalizeEPC("30A1B2"); !errors.Is(err, ErrInvalidEPC) {
		t.Errorf("short epc: err = %v, want ErrInvalidEPC", err)
	}
	// 非 hex。
	if _, err := NormalizeEPC("30ZZA1B2C3D4E5F60718293A"); !errors.Is(err, ErrInvalidEPC) {
		t.Errorf("non-hex: err = %v, want ErrInvalidEPC", err)
	}
	// 头部字节不在 30/32/33/35(31/34/FF 各验一)。
	for _, head := range []string{"31", "34", "FF"} {
		epc := head + "A1B2C3D4E5F60718293A4B"
		if _, err := NormalizeEPC(epc); !errors.Is(err, ErrInvalidEPC) {
			t.Errorf("head %s: err = %v, want ErrInvalidEPC", head, err)
		}
	}
}

func TestIdentityUpdateSemantics(t *testing.T) {
	if got := IdentityEffText("keep", nil); got != "keep" {
		t.Errorf("nil pointer should keep current, got %q", got)
	}
	if got := IdentityEffText(" old ", strPtr("")); got != "" {
		t.Errorf("empty string should clear, got %q", got)
	}
	if got := IdentityEffText("x", strPtr(" y ")); got != "y" {
		t.Errorf("trim expected, got %q", got)
	}
	if got, err := IdentityEffMac("AA:BB:CC:DD:EE:01", nil); err != nil || got != "AA:BB:CC:DD:EE:01" {
		t.Errorf("nil mac should keep, got %q err %v", got, err)
	}
	if got, err := IdentityEffMac("old", strPtr("AA-BB-CC-DD-EE-FF")); err != nil || got != "AA-BB-CC-DD-EE-FF" {
		t.Errorf("dash mac should apply, got %q err %v", got, err)
	}
	if _, err := IdentityEffMac("old", strPtr("bad")); !errors.Is(err, ErrInvalidMAC) {
		t.Errorf("bad mac: err = %v, want ErrInvalidMAC", err)
	}
}

func TestIdentityDuplicateMessage(t *testing.T) {
	e := &ErrAssetIdentityDuplicate{Field: "sn"}
	if got := e.Error(); got != "asset: sn already exists" {
		t.Errorf("message = %q, want field name inside", got)
	}
}

func strPtr(s string) *string { return &s }
