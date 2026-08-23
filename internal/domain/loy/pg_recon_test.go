package loy

import "testing"

func TestClassifyPointRecon(t *testing.T) {
	if got := classifyPointRecon(100, 100); got != ReconDiffMatch {
		t.Fatalf("match case got %s", got)
	}
	if got := classifyPointRecon(100, 90); got != ReconDiffBalanceDrift {
		t.Fatalf("drift case got %s", got)
	}
	if got := classifyPointRecon(0, 0); got != ReconDiffMatch {
		t.Fatalf("zero case got %s", got)
	}
}

func TestIdsOf(t *testing.T) {
	rows := []PointReconRow{{CustomerID: 7}, {CustomerID: 9}}
	ids := idsOf(rows)
	if len(ids) != 2 || ids[0] != 7 || ids[1] != 9 {
		t.Fatalf("ids=%v", ids)
	}
}
