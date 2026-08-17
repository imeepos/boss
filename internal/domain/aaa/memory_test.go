package aaa

import (
	"context"
	"testing"
)

func TestMemoryAuthorizerDecide(t *testing.T) {
	auth := NewMemoryAuthorizer([]Profile{
		{LOID: "active", Status: StatusActive, Bandwidth: "100M/50M", SessionTTL: 60},
		{LOID: "suspended", Status: StatusSuspended},
	})

	tests := []struct {
		name    string
		loid    string
		wantOK  bool
		wantErr error
		wantBw  string
	}{
		{"active", "active", true, nil, "100M/50M"},
		{"suspended", "suspended", false, ErrSuspended, ""},
		{"missing", "nope", false, ErrNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := auth.Decide(context.Background(), tt.loid)
			if err != tt.wantErr {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if dec.Authorize != tt.wantOK {
				t.Fatalf("Authorize = %v, want %v", dec.Authorize, tt.wantOK)
			}
			if tt.wantBw != "" && dec.Bandwidth != tt.wantBw {
				t.Fatalf("Bandwidth = %q, want %q", dec.Bandwidth, tt.wantBw)
			}
		})
	}
}
