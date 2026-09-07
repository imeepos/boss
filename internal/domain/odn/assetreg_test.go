package odn

import (
	"errors"
	"testing"
)

func TestValidateRegistrationInput(t *testing.T) {
	cases := []struct {
		name         string
		entityKind   string
		sourceKind   string
		facilityCode string
		deviceID     int64
		projectID    int64
		value        float64
		wantErr      bool
	}{
		{"facility procurement ok", RegFacility, RegSourceProcurement, "P00001", 0, 0, 100, false},
		{"device construction ok", RegDevice, RegSourceConstruction, "", 7, 3, 0, false},
		{"direct with project rejected", RegFacility, RegSourceDirect, "P00001", 0, 3, 1, true},
		{"construction without project rejected", RegDevice, RegSourceConstruction, "", 7, 0, 1, true},
		{"facility missing code rejected", RegFacility, RegSourceDirect, "", 0, 0, 0, true},
		{"facility with device rejected", RegFacility, RegSourceDirect, "P00001", 7, 0, 0, true},
		{"device without id rejected", RegDevice, RegSourceDirect, "", 0, 0, 0, true},
		{"negative value rejected", RegFacility, RegSourceDirect, "P00001", 0, 0, -1, true},
		{"unknown entity rejected", "SITE", RegSourceDirect, "X", 0, 0, 0, true},
		{"unknown source rejected", RegFacility, "IMPORT", "P00001", 0, 0, 0, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRegistrationInput(tc.entityKind, tc.sourceKind, tc.facilityCode, tc.deviceID, tc.projectID, tc.value)
			if tc.wantErr && !errors.Is(err, ErrRegInput) {
				t.Fatalf("want ErrRegInput, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want ok, got %v", err)
			}
		})
	}
}

func TestValidRegistrationNo(t *testing.T) {
	if !ValidRegistrationNo("ZG-20260907-00001") {
		t.Fatal("want valid ZG no")
	}
	if ValidRegistrationNo("ZG-20260907-1") || ValidRegistrationNo("ST-20260907-00001") {
		t.Fatal("want invalid no rejected")
	}
}
