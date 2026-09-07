package odn

import (
	"math"
	"testing"
)

// costPerServed 口径单元测试(fields.md 1.5.11:分母 0 或成本未登记一律未登记)。
func TestCostPerServed(t *testing.T) {
	t.Run("成本未登记返回nil", func(t *testing.T) {
		if got := costPerServed(nil, 5); got != nil {
			t.Fatalf("期望 nil,得 %v", *got)
		}
	})
	t.Run("分母为0返回nil", func(t *testing.T) {
		v := 120.0
		if got := costPerServed(&v, 0); got != nil {
			t.Fatalf("期望 nil,得 %v", *got)
		}
	})
	t.Run("正常相除", func(t *testing.T) {
		v := 100.0
		got := costPerServed(&v, 8)
		if got == nil || math.Abs(*got-12.5) > 1e-9 {
			t.Fatalf("期望 12.5,得 %v", got)
		}
	})
}
