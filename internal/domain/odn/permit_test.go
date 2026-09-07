package odn

import "testing"

// 许可单状态机与开工门控纯函数单测(000211,不依赖 DB)。

func TestPermitInitialStatus(t *testing.T) {
	st, err := PermitInitialStatus(PermitKindROW)
	if err != nil || st != PRowNotStarted {
		t.Fatalf("ROW initial=%s err=%v", st, err)
	}
	st, err = PermitInitialStatus(PermitKindPECE)
	if err != nil || st != PPecePendingSign {
		t.Fatalf("PECE initial=%s err=%v", st, err)
	}
	if _, err := PermitInitialStatus("XXX"); err == nil {
		t.Fatal("unknown kind should error")
	}
}

func TestValidatePermitTransitionROW(t *testing.T) {
	legal := [][2]string{
		{PRowNotStarted, PRowPending}, {PRowPending, PRowApproved}, {PRowPending, PRowNotStarted},
		{PRowApproved, PRowExpired}, {PRowExpired, PRowPending},
		{PRowNotStarted, PermitNA}, {PermitNA, PRowNotStarted},
	}
	for _, tc := range legal {
		if err := ValidatePermitTransition(PermitKindROW, tc[0], tc[1]); err != nil {
			t.Fatalf("ROW %s to %s should be legal: %v", tc[0], tc[1], err)
		}
	}
	illegal := [][2]string{
		{PRowNotStarted, PRowApproved}, {PRowApproved, PRowPending},
		{PRowExpired, PRowApproved}, {PRowExpired, PermitNA}, {PRowPending, PRowExpired},
	}
	for _, tc := range illegal {
		if err := ValidatePermitTransition(PermitKindROW, tc[0], tc[1]); err == nil {
			t.Fatalf("ROW %s to %s should be illegal", tc[0], tc[1])
		}
	}
}

func TestValidatePermitTransitionPECE(t *testing.T) {
	legal := [][2]string{
		{PPecePendingSign, PPeceSigned}, {PPeceSigned, PPeceStamped},
		{PPeceSigned, PPecePendingSign}, {PPecePendingSign, PermitNA}, {PermitNA, PPecePendingSign},
	}
	for _, tc := range legal {
		if err := ValidatePermitTransition(PermitKindPECE, tc[0], tc[1]); err != nil {
			t.Fatalf("PECE %s to %s should be legal: %v", tc[0], tc[1], err)
		}
	}
	if err := ValidatePermitTransition(PermitKindPECE, PPecePendingSign, PPeceStamped); err == nil {
		t.Fatal("PECE skip sign should be illegal")
	}
	if err := ValidatePermitTransition(PermitKindROW, PRowNotStarted, PPeceSigned); err == nil {
		t.Fatal("cross-kind state should be illegal")
	}
}
func mkPermit(kind, status, validUntil string) Permit {
	return Permit{Kind: kind, Status: status, ValidUntil: validUntil}
}

func TestEvaluatePermitGate(t *testing.T) {
	today := "2026-09-07"
	if rep := EvaluatePermitGate(nil, today); rep.OK || len(rep.Problems) != 2 {
		t.Fatalf("empty permits should fail with 2 problems: %+v", rep)
	}
	rowOK := mkPermit(PermitKindROW, PRowApproved, "2026-12-31")
	peceOK := mkPermit(PermitKindPECE, PPeceStamped, "")
	if rep := EvaluatePermitGate([]Permit{rowOK, peceOK}, today); !rep.OK {
		t.Fatalf("valid permits should pass: %+v", rep)
	}
	expired := mkPermit(PermitKindROW, PRowApproved, "2026-01-01")
	rep := EvaluatePermitGate([]Permit{expired, peceOK}, today)
	if rep.OK || len(rep.ExpiredIDs) == 0 {
		t.Fatalf("expired ROW should block: %+v", rep)
	}
	na := EvaluatePermitGate([]Permit{mkPermit(PermitKindROW, PermitNA, ""), mkPermit(PermitKindPECE, PermitNA, "")}, today)
	if !na.OK {
		t.Fatalf("NA should satisfy: %+v", na)
	}
	if rep := EvaluatePermitGate([]Permit{rowOK, mkPermit(PermitKindPECE, PPeceSigned, "")}, today); rep.OK {
		t.Fatal("signed-only PECE should not pass")
	}
	edge := EvaluatePermitGate([]Permit{mkPermit(PermitKindROW, PRowApproved, today), peceOK}, today)
	if !edge.OK {
		t.Fatalf("validUntil == today should pass: %+v", edge)
	}
}
