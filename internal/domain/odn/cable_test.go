package odn

import "testing"

// TestOrderEndpoints 规范 5.2 A 端方向优先级:SNW>ODF>OCC>CLS>ODB>P>MH。
func TestOrderEndpoints(t *testing.T) {
	cases := []struct {
		c1, c2, wantA, wantB string
		wantErr              error
	}{
		// 规范 5.1 示例:ODF001--OCC001 → ODF 为 A 端。
		{"ODF001", "OCC001", "ODF001", "OCC001", nil},
		// 顺序颠倒输入也应定向为 ODF 在前。
		{"OCC001", "ODF001", "ODF001", "OCC001", nil},
		// 电杆到电杆示例:P01001--P02005 同优先级,规范未定义,报错。
		{"P01001", "P02005", "", "", ErrSamePriority},
		// CLS(4) > P(6)。
		{"P01001", "CLS00001", "CLS00001", "P01001", nil},
		// ODB(5) > MH(7)。
		{"MH01001", "ODB001", "ODB001", "MH01001", nil},
		// SNW(1) 最高。
		{"SNW001", "P01001", "SNW001", "P01001", nil},
		// 非法端点。
		{"P01001", "XX001", "", "", ErrInvalidEndpoint},
		{"", "P01001", "", "", ErrInvalidEndpoint},
	}
	for _, c := range cases {
		a, b, err := OrderEndpoints(c.c1, c.c2)
		if c.wantErr != nil {
			if err != c.wantErr {
				t.Errorf("%s-%s: 期望 %v,实际 %v", c.c1, c.c2, c.wantErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s-%s: 意外错误 %v", c.c1, c.c2, err)
			continue
		}
		if a != c.wantA || b != c.wantB {
			t.Errorf("%s-%s: A=%s B=%s,期望 A=%s B=%s", c.c1, c.c2, a, b, c.wantA, c.wantB)
		}
	}
}
