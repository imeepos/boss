package userdata

import (
	"errors"
	"strings"
	"testing"
)

// AssertListContract 契约守护回归(postmortem 0002):
// 一旦后端 SQL 漏 AS 出主键列或值被 NULL,前端看到 undefined 死链。
// 这里锁住守卫的命中条件,防止规则被悄悄放宽。
func TestAssertListContract(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		pk      string
		items   []map[string]any
		wantErr bool
	}{
		{"空列表放行", "ListX", "addonId", nil, false},
		{"空 map 列表放行", "ListX", "addonId", []map[string]any{}, false},
		{"主键列存在且非零", "ListAddons", "addonId",
			[]map[string]any{{"addonId": "ADD-01", "name": "x"}}, false},
		{"主键缺失整列", "ListAddons", "addonId",
			[]map[string]any{{"name": "x"}}, true},
		{"主键存在但 nil", "ListCoupons", "couponId",
			[]map[string]any{{"couponId": nil}}, true},
		{"主键空串", "ListCoupons", "couponId",
			[]map[string]any{{"couponId": ""}}, true},
		{"主键为 0(int64)", "ListUserFaqs", "faqId",
			[]map[string]any{{"faqId": int64(0)}}, true},
		{"第二行漂移也命中", "ListCoupons", "couponId",
			[]map[string]any{
				{"couponId": "CPN-01"},
				{"couponId": ""},
			}, true},
		{"全部主键非零通过", "ListCoupons", "couponId",
			[]map[string]any{
				{"couponId": "CPN-01"},
				{"couponId": "CPN-02"},
			}, false},
		{"int 类型主键非零也通过", "ListInviteConfig", "id",
			[]map[string]any{{"id": int(1)}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := AssertListContract(tc.op, tc.pk, tc.items)
			if tc.wantErr && err == nil {
				t.Fatalf("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
			if err != nil {
				if !errors.Is(err, ErrContractDrift) {
					t.Fatalf("err=%v 不被 ErrContractDrift 识别", err)
				}
				if !strings.Contains(err.Error(), tc.op) {
					t.Fatalf("err 缺 op=%q: %v", tc.op, err)
				}
				if !strings.Contains(err.Error(), tc.pk) {
					t.Fatalf("err 缺 pk=%q: %v", tc.pk, err)
				}
			}
		})
	}
}

func TestErrContractDrift_Is(t *testing.T) {
	wrapped := &contractDriftError{op: "x", field: "y"}
	if !errors.Is(wrapped, ErrContractDrift) {
		t.Fatalf("errors.Is 必须命中 ErrContractDrift")
	}
}