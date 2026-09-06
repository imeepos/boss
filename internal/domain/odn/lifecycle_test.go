package odn

import "testing"

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
