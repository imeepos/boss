package odn

import (
	"errors"
	"testing"
)

// 应付台账纯逻辑单测(P-INFRA-1 W6/F8):状态派生与付款/核减校验口径。
func TestComputePayableStatus(t *testing.T) {
	cases := []struct {
		payable, deducted, paid float64
		want                    string
	}{
		{1000, 0, 0, APOpen},
		{1000, 0, 400, APPartial},
		{1000, 0, 1000, APPaid},
		{1000, 0, 1200, APPaid},  // 超付也算付清(防御,登记层已拦)
		{1000, 600, 400, APPaid}, // 核减后净应付 400,已付 400
		{1000, 600, 399.99, APPartial},
		{1000, 200, 0, APOpen},
	}
	for _, c := range cases {
		if got := ComputePayableStatus(c.payable, c.deducted, c.paid); got != c.want {
			t.Errorf("ComputePayableStatus(%v,%v,%v)=%s want %s", c.payable, c.deducted, c.paid, got, c.want)
		}
	}
}

func TestValidatePayRegister(t *testing.T) {
	if err := ValidatePayRegister(PayTransfer, 400, 600); err != nil {
		t.Errorf("合法分期付款被拒: %v", err)
	}
	if err := ValidatePayRegister(PayCash, 0, 600); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("0 金额应 42200: %v", err)
	}
	if err := ValidatePayRegister(PayTransfer, -1, 600); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("负金额应 42200: %v", err)
	}
	if err := ValidatePayRegister("WECHAT", 100, 600); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("非法方式应 42200: %v", err)
	}
	if err := ValidatePayRegister(PayTransfer, 600.01, 600); !errors.Is(err, ErrPayableState) {
		t.Errorf("超余额应 40900: %v", err)
	}
	// 分期:多笔累计到余额耗尽即 PAID,不允许再登记。
	if err := ValidatePayRegister(PayTransfer, 0.01, 0); !errors.Is(err, ErrPayableState) {
		t.Errorf("余额耗尽再登记应 40900: %v", err)
	}
}

func TestValidateDeduct(t *testing.T) {
	// 合法:应付 1000,已付 400,核减 200 → 净应付 800 >= 已付 400。
	if err := ValidateDeduct("复审核减", 200, 1000, 0, 400); err != nil {
		t.Errorf("合法核减被拒: %v", err)
	}
	if err := ValidateDeduct("", 200, 1000, 0, 400); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("缺原因应 42200: %v", err)
	}
	if err := ValidateDeduct("x", 0, 1000, 0, 400); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("0 金额应 42200: %v", err)
	}
	if err := ValidateDeduct("x", 1200, 1000, 0, 0); !errors.Is(err, ErrPayableState) {
		t.Errorf("核减超应付应 40900: %v", err)
	}
	// 核减后净应付 400 < 已付 500 → 防超付拒绝。
	if err := ValidateDeduct("x", 700, 1000, 0, 500); !errors.Is(err, ErrPayableState) {
		t.Errorf("核减低于已付应 40900: %v", err)
	}
	// 累计核减口径:已核 600 再核 400 恰好等于应付,已付 0 放行。
	if err := ValidateDeduct("x", 400, 1000, 600, 0); err != nil {
		t.Errorf("累计核减到应付上限应放行: %v", err)
	}
}

func TestRound2(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0.1 + 0.2, 0.3},
		{400.00, 400},
		{0.005, 0.01},
	}
	for _, c := range cases {
		if got := round2(c.in); got != c.want {
			t.Errorf("round2(%v)=%v want %v", c.in, got, c.want)
		}
	}
}
