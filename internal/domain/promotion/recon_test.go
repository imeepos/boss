package promotion

import "testing"

func TestClassifyCouponRecon(t *testing.T) {
	cases := []struct {
		name string
		row  CouponReconRow
		want string
	}{
		{"match", CouponReconRow{
			IssuedQty: 10, ActualIssued: 10, UsedCount: 3, RedemptionCnt: 3,
		}, ReconDiffMatch},
		{"counter drift", CouponReconRow{
			IssuedQty: 10, ActualIssued: 9, UsedCount: 3, RedemptionCnt: 3,
		}, ReconDiffCounterDrift},
		{"redemption lost", CouponReconRow{
			IssuedQty: 10, ActualIssued: 10, UsedCount: 3, RedemptionCnt: 2,
		}, ReconDiffRedemptionLost},
		{"counter drift 优先于 redemption", CouponReconRow{
			IssuedQty: 10, ActualIssued: 9, UsedCount: 3, RedemptionCnt: 2,
		}, ReconDiffCounterDrift},
	}
	for _, tc := range cases {
		if got := classifyCouponRecon(tc.row); got != tc.want {
			t.Fatalf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func TestCouponReconSummaryAggregates(t *testing.T) {
	rows := []CouponReconRow{
		{ActualIssued: 10, UsedCount: 2, RedeemedAmount: 500, FaceValueTotal: 800,
			DiffKind: ReconDiffMatch},
		{ActualIssued: 5, UsedCount: 5, RedeemedAmount: 700, FaceValueTotal: 0,
			DiffKind: ReconDiffCounterDrift},
	}
	sum := CouponReconSummary{Templates: len(rows), ByDiff: map[string]int{}}
	for _, r := range rows {
		sum.ActualIssued += r.ActualIssued
		sum.UsedCount += r.UsedCount
		sum.RedeemedAmount += r.RedeemedAmount
		sum.FaceValueTotal += r.FaceValueTotal
		sum.ByDiff[r.DiffKind]++
	}
	if sum.ActualIssued != 15 || sum.UsedCount != 7 || sum.RedeemedAmount != 1200 ||
		sum.FaceValueTotal != 800 || sum.ByDiff[ReconDiffMatch] != 1 ||
		sum.ByDiff[ReconDiffCounterDrift] != 1 {
		t.Fatalf("summary=%+v", sum)
	}
}
