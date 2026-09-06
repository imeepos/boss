package odn

import "testing"

// TestValidateProjectTransition 施工单状态机:线性推进,同态 no-op,回退/终态再动拒绝。
func TestValidateProjectTransition(t *testing.T) {
	cases := []struct {
		from, to string
		wantErr  bool
	}{
		{CPending, CBuilding, false},
		{CBuilding, CAccepted, false},
		{CPending, CPending, false},
		{CAccepted, CAccepted, false},
		{CPending, CAccepted, true},
		{CBuilding, CPending, true},
		{CAccepted, CBuilding, true},
		{CAccepted, CPending, true},
		{"UNKNOWN", CPending, true},
		{CPending, "UNKNOWN", true},
	}
	for _, c := range cases {
		err := ValidateProjectTransition(c.from, c.to)
		if c.wantErr && err == nil {
			t.Errorf("%s→%s 期望拒绝,实际通过", c.from, c.to)
		}
		if !c.wantErr && err != nil {
			t.Errorf("%s→%s 意外错误 %v", c.from, c.to, err)
		}
	}
}
