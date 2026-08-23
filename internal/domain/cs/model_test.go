package cs

import "testing"

func TestMetricNamesAreStable(t *testing.T) {
	got := []string{FirstResponseRate, FirstContactResolutionRate, SLAOverdueRate, RepeatTicketRate, ArrearsRecoveryRate}
	if len(got) != 5 || got[0] != "first_response_rate" || got[4] != "arrears_recovery_rate" {
		t.Fatalf("metric keys=%v", got)
	}
}
