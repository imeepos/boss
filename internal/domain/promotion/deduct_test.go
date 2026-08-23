package promotion

import "testing"

func TestDeductAmount(t *testing.T) {
	cases := []struct {
		name              string
		cType             string
		face, thr, max, b int64
		want              int64
	}{
		{"现金券足额", TypeCash, 2000, 0, 0, 9900, 2000},
		{"现金券超账单封顶", TypeCash, 20000, 0, 0, 9900, 9900},
		{"满减达门槛", TypeFullCut, 2000, 10000, 0, 15000, 2000},
		{"满减未达门槛", TypeFullCut, 2000, 10000, 0, 9900, -1},
		{"折扣85折", TypeDiscout, 8500, 0, 0, 10000, 1500},
		{"折扣封顶", TypeDiscout, 8500, 0, 500, 10000, 500},
	}
	for _, tc := range cases {
		if got := deductAmount(tc.cType, tc.face, tc.thr, tc.max, tc.b); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestCouponView(t *testing.T) {
	avail := Coupon{CouponID: "C1", Status: "ISSUED", Type: TypeCash, FaceValue: 1000}
	item, ok := couponView(avail, "available", 5000)
	if !ok || item["estDeduct"] != int64(1000) {
		t.Fatalf("available item=%v ok=%v", item, ok)
	}
	// 门槛不满足:不进可用列表
	_, ok = couponView(Coupon{Status: "ISSUED", Type: TypeFullCut, Threshold: 10000}, "available", 5000)
	if ok {
		t.Fatal("threshold-unmet coupon should be filtered")
	}
	// 转赠中仅 all 视图可见
	_, ok = couponView(Coupon{Status: "ISSUED", Code: "GFT-1"}, "available", 0)
	if ok {
		t.Fatal("gifting coupon not available")
	}
	item, ok = couponView(Coupon{Status: "ISSUED", Code: "GFT-1"}, "all", 0)
	if !ok || item["status"] != "gifting" {
		t.Fatalf("all view item=%v", item)
	}
}

// TestCouponView_FilterCaseInsensitive used/USED 过滤大小写不敏感(000119 验收集发现)。
func TestCouponView_FilterCaseInsensitive(t *testing.T) {
	used := Coupon{Status: "USED"}
	if _, ok := couponView(used, "used", 0); !ok {
		t.Fatal("status=used 过滤应命中 USED 券")
	}
	if _, ok := couponView(used, "USED", 0); !ok {
		t.Fatal("status=USED 过滤应命中 USED 券")
	}
	if _, ok := couponView(used, "available", 0); ok {
		t.Fatal("used 券不应出现在 available 过滤")
	}
	item, ok := couponView(used, "all", 0)
	if !ok || item["status"] != "used" {
		t.Fatalf("展示态应小写化: %#v", item)
	}
}
