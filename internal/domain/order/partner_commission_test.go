package order

import (
	"context"
	"testing"
)

type commissionParams struct {
	value string
	err   error
}

func (p commissionParams) GetParam(context.Context, string) (string, error) { return p.value, p.err }

func TestPartnerRate(t *testing.T) {
	cases := []struct {
		name   string
		params PartnerCommissionRate
		want   float64
	}{
		{"default when nil", nil, 0.10},
		{"configured", commissionParams{value: "0.25"}, 0.25},
		{"invalid fallback", commissionParams{value: "bad"}, 0.10},
		{"out of range fallback", commissionParams{value: "1.01"}, 0.10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := partnerRate(context.Background(), tc.params); got != tc.want {
				t.Fatalf("rate=%v want %v", got, tc.want)
			}
		})
	}
}
