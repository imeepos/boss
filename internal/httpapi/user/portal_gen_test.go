package userapi

import "testing"

// 回归:pgx 把 bigint 主键扫成 int64,toStr 必须回填字符串 ID(2026-08-24 E2E 发现 /addresses addressId 恒空)。
func TestToStrNumericIDs(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{int64(1), "1"},
		{int(42), "42"},
		{"abc", "abc"},
		{"", ""},
		{3.14, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := toStr(c.in); got != c.want {
			t.Errorf("toStr(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
