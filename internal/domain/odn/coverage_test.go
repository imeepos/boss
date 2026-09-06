package odn

import "testing"

// TestValidateCoverage 状态/目标一致性(fields.md 1.5.7 + 库端 target_chk 兜底)。
func TestValidateCoverage(t *testing.T) {
	cases := []struct {
		name     string
		status   string
		facility string
		device   int64
		wantErr  bool
	}{
		{"SERVED 挂设施", CovServed, "ODB001", 0, false},
		{"SERVED 挂设备", CovServed, "", 42, false},
		{"SERVED 双挂", CovServed, "SDB001", 42, false},
		{"PENDING 挂设施", CovPending, "CLS00007", 0, false},
		{"SERVED 空目标拒", CovServed, "", 0, true},
		{"PENDING 空目标拒", CovPending, "", 0, true},
		{"UNSERVED 允许全空", CovUnserved, "", 0, false},
		{"未知状态拒", "FULL", "ODB001", 0, true},
		{"空状态拒", "", "", 0, true},
	}
	for _, c := range cases {
		err := ValidateCoverage(c.status, c.facility, c.device)
		if c.wantErr && err == nil {
			t.Errorf("%s: 期望拒绝,实际通过(%s/%s/%d)", c.name, c.status, c.facility, c.device)
		}
		if !c.wantErr && err != nil {
			t.Errorf("%s: 意外错误 %v", c.name, err)
		}
	}
}

// TestHaversineM 马尼拉已知距离粗校验(同点 0,约 0.01 度纬差 ~1.1km)。
func TestHaversineM(t *testing.T) {
	if d := haversineM(14.6, 121.0, 14.6, 121.0); d > 0.001 {
		t.Errorf("同点距离应为 0,实际 %f", d)
	}
	d := haversineM(14.60, 121.00, 14.61, 121.00) // 纬差 0.01 度
	if d < 1000 || d > 1200 {
		t.Errorf("0.01 度纬差应约 1.1km,实际 %f", d)
	}
}

// TestResolveStatus 半径阈值:2km 内可装,超出未覆盖。
func TestResolveStatus(t *testing.T) {
	if got := ResolveStatus(1999); got != CovServed {
		t.Errorf("1999m 应 SERVED,实际 %s", got)
	}
	if got := ResolveStatus(ResolveRadiusM); got != CovServed {
		t.Errorf("边界 %dm 应 SERVED,实际 %s", ResolveRadiusM, got)
	}
	if got := ResolveStatus(2001); got != CovUnserved {
		t.Errorf("2001m 应 UNSERVED,实际 %s", got)
	}
}
