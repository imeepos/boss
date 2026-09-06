package odn

import "testing"

// TestValidatePortTransition 端口状态机:空闲可预占/在网,预占可开通可释放,在网仅可拆机释放。
func TestValidatePortTransition(t *testing.T) {
	cases := []struct {
		from, to string
		wantErr  bool
	}{
		{PortIdle, PortReserved, false},
		{PortIdle, PortInSvc, false},
		{PortReserved, PortInSvc, false},
		{PortReserved, PortIdle, false},
		{PortInSvc, PortIdle, false},
		{PortReserved, PortReserved, false},
		{PortReserved, PortIdle + PortIdle, true}, // 未知目标态
		{PortInSvc, PortReserved, true},           // 在网不可回预占
		{"UNKNOWN", PortIdle, true},
		{PortIdle, "UNKNOWN", true},
	}
	for _, c := range cases {
		err := ValidatePortTransition(c.from, c.to)
		if c.wantErr && err == nil {
			t.Errorf("%s→%s 期望拒绝,实际通过", c.from, c.to)
		}
		if !c.wantErr && err != nil {
			t.Errorf("%s→%s 意外错误 %v", c.from, c.to, err)
		}
	}
}
