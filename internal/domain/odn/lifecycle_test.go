package odn

import (
	"errors"
	"testing"
)

// TestValidateLifecycleTransition 状态机规则:线性推进+各态可退役,RETIRED 终态,同态 no-op。
func TestValidateLifecycleTransition(t *testing.T) {
	cases := []struct {
		from, to string
		wantErr  bool
	}{
		{LCPlanned, LCInBuild, false},
		{LCPlanned, LCInService, false},
		{LCPlanned, LCRetired, false},
		{LCInBuild, LCInService, false},
		{LCInBuild, LCRetired, false},
		{LCInService, LCRetired, false},
		{LCPlanned, LCPlanned, false},
		{LCRetired, LCRetired, false},
		{LCInBuild, LCPlanned, true},
		{LCInService, LCInBuild, true},
		{LCInService, LCPlanned, true},
		{LCRetired, LCInService, true},
		{LCRetired, LCPlanned, true},
		{"UNKNOWN", LCInService, true},
		{LCPlanned, "UNKNOWN", true},
	}
	for _, c := range cases {
		err := ValidateLifecycleTransition(c.from, c.to)
		if c.wantErr && err == nil {
			t.Errorf("%s→%s 期望拒绝,实际通过", c.from, c.to)
		}
		if !c.wantErr && err != nil {
			t.Errorf("%s→%s 意外错误 %v", c.from, c.to, err)
		}
	}
}

// TestNormalizeCreateLifecycle 新建初始态口径:留空补 PLANNED(与导入规则 ④ 一致),
// 显式 IN_BUILD/IN_SERVICE 放行(登记既有在网设施),RETIRED 与未知值拒绝。
func TestNormalizeCreateLifecycle(t *testing.T) {
	cases := []struct {
		in, want string
		wantErr  bool
	}{
		{"", LCPlanned, false},
		{LCPlanned, LCPlanned, false},
		{LCInBuild, LCInBuild, false},
		{LCInService, LCInService, false},
		{LCRetired, "", true},
		{"UNKNOWN", "", true},
		{"planned", "", true},
	}
	for _, c := range cases {
		got, err := NormalizeCreateLifecycle(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q 期望拒绝,实际得 %q", c.in, got)
			} else if !errors.Is(err, ErrInvalidLifecycle) {
				t.Errorf("%q 错误未包装 ErrInvalidLifecycle: %v", c.in, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q 意外错误 %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("%q 得 %q 期望 %q", c.in, got, c.want)
		}
	}
}
