package odn

import (
	"errors"
	"testing"
)

// 结算单状态机单测(P-INFRA-1 W1):合法转移放行,回退/终态再动拒绝,同态 no-op。
func TestValidateSettlementTransition(t *testing.T) {
	legal := []struct{ from, to string }{
		{SPending, SSettled},
		{SPending, SVoided},
		{SSettled, SVoided},
	}
	for _, c := range legal {
		if err := ValidateSettlementTransition(c.from, c.to); err != nil {
			t.Errorf("%s->%s: got %v want nil", c.from, c.to, err)
		}
	}
	illegal := []struct{ from, to string }{
		{SSettled, SPending}, // 已结算不可回退
		{SVoided, SPending},  // 终态不可再流转(重开以新单表达)
		{SVoided, SSettled},
	}
	for _, c := range illegal {
		if err := ValidateSettlementTransition(c.from, c.to); !errors.Is(err, ErrSettlementState) {
			t.Errorf("%s->%s: got %v want ErrSettlementState", c.from, c.to, err)
		}
	}
	for _, s := range []string{SPending, SSettled, SVoided} {
		if err := ValidateSettlementTransition(s, s); err != nil {
			t.Errorf("同态 %s: got %v want nil", s, err)
		}
	}
}
