package odn

import "testing"

// TestValidateFacilityCode 红线 3:5 位数字网格分区模式(规范 4.2)。
func TestValidateFacilityCode(t *testing.T) {
	cases := []struct {
		code     string
		wantKind string
		wantGrid int16
		wantSeq  int
		wantErr  bool
	}{
		{"P01001", KindPole, 1, 1, false},
		{"MH01001", KindManhole, 1, 1, false},
		{"P99099", KindPole, 99, 99, false},
		{"TW00001", KindTower, 0, 1, false},
		{"CLS00042", KindClosure, 0, 42, false},
		{"TBX00100", KindTermBox, 0, 100, false},
		{"TW99999", KindTower, 0, 99999, false},
		// 非法:4 位顺序模式(规范 4.2 强制废弃)、位数错、序号 000 预留、未知前缀、空。
		{"P0100", "", 0, 0, true},    // 位数不足
		{"MH010011", "", 0, 0, true}, // 位数超
		{"P01000", "", 0, 0, true},   // 序号 000 预留禁用(规范 4.1)
		{"TW00000", "", 0, 0, true},  // 顺序号 00000 预留
		{"XX01001", "", 0, 0, true},  // 未定义前缀
		{"p01001", "", 0, 0, true},   // 小写
		{"", "", 0, 0, true},
	}
	for _, c := range cases {
		kind, grid, seq, err := ValidateFacilityCode(c.code)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: 期望非法,实际通过(kind=%s)", c.code, kind)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 意外错误 %v", c.code, err)
			continue
		}
		if kind != c.wantKind || grid != c.wantGrid || seq != c.wantSeq {
			t.Errorf("%s: 解析 kind=%s grid=%d seq=%d,期望 %s/%d/%d",
				c.code, kind, grid, seq, c.wantKind, c.wantGrid, c.wantSeq)
		}
	}
}
