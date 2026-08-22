package adminapi

import (
	"testing"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

func TestSummarizeAAA(t *testing.T) {
	got := summarizeAAA(
		[]aaa.LoAccount{{Status: "ACTIVE"}, {Status: "SUSPENDED"}, {Status: "CLOSED"}},
		[]aaa.CdrRecord{{BillingStatus: "UNBILLED"}, {BillingStatus: "BILLED"}},
		[]aaa.AuthLog{{Result: "SUCCESS"}, {Result: "FAILED"}, {Result: "SUCCESS"}},
	)
	want := map[string]any{
		"accounts": 3, "active": 1, "suspended": 1, "closed": 1,
		"cdrs": 2, "unbilled": 1, "authSuccess": 2, "authFailed": 1,
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("%s=%v, want %v", key, got[key], value)
		}
	}
}
