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

// W5 成本与户级口径单元测试。
func TestSumCostPtr(t *testing.T) {
	t.Run("全nil为nil", func(t *testing.T) {
		if got := sumCostPtr(nil, nil, nil); got != nil {
			t.Fatalf("期望 nil,得 %v", *got)
		}
	})
	t.Run("部分nil按0计入", func(t *testing.T) {
		a, c := 100.0, 30.5
		got := sumCostPtr(&a, nil, &c)
		if got == nil || math.Abs(*got-130.5) > 1e-9 {
			t.Fatalf("期望 130.5,得 %v", got)
		}
	})
}

func TestCostPerHomes(t *testing.T) {
	t.Run("潜在户数缺失返回nil", func(t *testing.T) {
		v := 100.0
		if got := costPerHomes(&v, nil); got != nil {
			t.Fatalf("期望 nil,得 %v", *got)
		}
	})
	t.Run("潜在户数为0返回nil", func(t *testing.T) {
		v := 100.0
		zero := 0
		if got := costPerHomes(&v, &zero); got != nil {
			t.Fatalf("期望 nil,得 %v", *got)
		}
	})
	t.Run("正常相除", func(t *testing.T) {
		v := 240.0
		n := 64
		got := costPerHomes(&v, &n)
		if got == nil || math.Abs(*got-3.75) > 1e-9 {
			t.Fatalf("期望 3.75,得 %v", got)
		}
	})
}

func TestHomesPotential(t *testing.T) {
	t.Run("二级器计户", func(t *testing.T) {
		p, c := homesPotential(2, false, 8, 3)
		if p != 8 || c != 3 {
			t.Fatalf("期望 8/3,得 %d/%d", p, c)
		}
	})
	t.Run("无二级链的一级器直达户", func(t *testing.T) {
		p, c := homesPotential(1, false, 32, 5)
		if p != 32 || c != 5 {
			t.Fatalf("期望 32/5,得 %d/%d", p, c)
		}
	})
	t.Run("下挂二级链的一级器端口不到户", func(t *testing.T) {
		p, c := homesPotential(1, true, 8, 2)
		if p != 0 || c != 0 {
			t.Fatalf("期望 0/0,得 %d/%d", p, c)
		}
	})
}

func TestAddPtr(t *testing.T) {
	a, b := 10.0, 20.0
	if got := addPtr(&a, &b); got == nil || *got != 30 {
		t.Fatalf("期望 30,得 %v", got)
	}
	if got := addPtr(nil, nil); got != nil {
		t.Fatalf("期望 nil,得 %v", *got)
	}
	if got := addPtr(nil, &b); got == nil || *got != 20 {
		t.Fatalf("期望 20,得 %v", got)
	}
}

func TestSortCityRows(t *testing.T) {
	rows := []CityInvestmentRow{
		{PrvCode: "PHL001", CityPrefix: "MNL"},
		{PrvCode: "PHL001", CityPrefix: "CEB"},
		{PrvCode: "PHL001", CityPrefix: "MNL"},
		{PrvCode: "PHL002", CityPrefix: "AAA"},
	}
	sortCityRows(rows)
	want := []string{"CEB", "MNL", "MNL", "AAA"}
	for i, w := range want {
		if rows[i].CityPrefix != w {
			t.Fatalf("第 %d 行期望 %s 得 %s", i, w, rows[i].CityPrefix)
		}
	}
}
