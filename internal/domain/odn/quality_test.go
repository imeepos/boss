package odn

import (
	"errors"
	"testing"
)

func TestValidateQualityTest(t *testing.T) {
	good := QualityTest{ResourceType: "FACILITY", ResourceRef: "P01001", TestKind: TestOTDR, Result: TestPass}
	if err := ValidateQualityTest(good); err != nil {
		t.Errorf("good = %v, want nil", err)
	}
	bad := []QualityTest{
		{ResourceType: "WARE", ResourceRef: "P01001", TestKind: TestOTDR, Result: TestPass},
		{ResourceType: "FACILITY", ResourceRef: "", TestKind: TestOTDR, Result: TestPass},
		{ResourceType: "FACILITY", ResourceRef: "P01001", TestKind: "VDSL", Result: TestPass},
		{ResourceType: "FACILITY", ResourceRef: "P01001", TestKind: TestOTDR, Result: "MAYBE"},
		{ResourceType: "FACILITY", ResourceRef: "P01001", TestKind: TestOTDR, Result: TestPass, AttenuationDB: -1},
	}
	for i, tc := range bad {
		if err := ValidateQualityTest(tc); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("bad[%d] = %v, want ErrInvalidInput", i, err)
		}
	}
}

func TestValidateDefectOpenAndTransition(t *testing.T) {
	if err := ValidateDefectOpen(QualityDefect{Severity: "MAJOR", Description: "x"}); err != nil {
		t.Errorf("open major = %v, want nil", err)
	}
	if err := ValidateDefectOpen(QualityDefect{Severity: "HUGE", Description: "x"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("severity HUGE = %v, want ErrInvalidInput", err)
	}
	if err := ValidateDefectOpen(QualityDefect{Severity: "MINOR"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("empty desc = %v, want ErrInvalidInput", err)
	}
	oc := []struct{ from, to string }{
		{DefectOpen, DefectRectifying}, {DefectRectifying, DefectRectifying},
		{DefectOpen, DefectVerified}, {DefectRectifying, DefectVerified},
	}
	for _, c := range oc {
		if err := ValidateDefectTransition(c.from, c.to); err != nil {
			t.Errorf("%s→%s = %v, want nil", c.from, c.to, err)
		}
	}
	bad := []struct{ from, to string }{
		{DefectVerified, DefectRectifying}, {DefectVerified, DefectVerified},
		{DefectOpen, DefectOpen}, {DefectRectifying, DefectOpen},
	}
	for _, c := range bad {
		if err := ValidateDefectTransition(c.from, c.to); !errors.Is(err, ErrDefectState) {
			t.Errorf("%s→%s = %v, want ErrDefectState", c.from, c.to, err)
		}
	}
}
