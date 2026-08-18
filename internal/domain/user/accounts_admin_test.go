// 账号管理写操作纯校验用例(手机号;建号/改号主链路由 PG 集成测试覆盖)。
package user

import "testing"

func TestValidatePhone(t *testing.T) {
	cases := []struct {
		phone string
		ok    bool
	}{
		{"", true},
		{"13800001234", true},
		{"+63 917 000 0000", true},
		{"abc", false},
		{"1", false},
		{"0123456789012345678901234567890123456789", false},
	}
	for _, c := range cases {
		if err := validatePhone(c.phone); (err == nil) != c.ok {
			t.Fatalf("validatePhone(%q)=%v want ok=%v", c.phone, err, c.ok)
		}
	}
}
