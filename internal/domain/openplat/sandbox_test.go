// 沙箱样例与隔离测试(M4)。
package openplat

import (
	"encoding/json"
	"testing"
)

func TestSandboxOrderByNoCoversAllStates(t *testing.T) {
	states := map[string]bool{}
	for _, no := range SandboxOrderNos() {
		raw, ok := SandboxOrderByNo(no)
		if !ok {
			t.Fatalf("sample %s listed but not found", no)
		}
		var o struct {
			Status string `json:"status"`
			Stage  int8   `json:"stage"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			t.Fatalf("unmarshal %s: %v", no, err)
		}
		if o.Stage < 1 || o.Stage > 12 {
			t.Fatalf("sample %s stage %d out of 1..12", no, o.Stage)
		}
		states[o.Status] = true
	}
	for _, s := range []string{"PENDING", "RESERVED", "INSTALLING", "DONE"} {
		if !states[s] {
			t.Fatalf("sample set missing status %s", s)
		}
	}
}

func TestSandboxOrderUnknownNo(t *testing.T) {
	if _, ok := SandboxOrderByNo("SBX-ORD-9999"); ok {
		t.Fatal("unknown sample must not resolve")
	}
	if _, ok := SandboxOrderByNo("ORD-20250817-001"); ok {
		t.Fatal("production order no must not resolve in sandbox set")
	}
}

func TestIsSandboxOrderNo(t *testing.T) {
	cases := map[string]bool{
		"SBX-ORD-0001":     true,
		"SBX-X":            true,
		"ORD-20250817-001": false,
		"SBX":              false,
		"":                 false,
	}
	for no, want := range cases {
		if got := IsSandboxOrderNo(no); got != want {
			t.Fatalf("IsSandboxOrderNo(%q)=%v want %v", no, got, want)
		}
	}
}

func TestSandboxWebhookSampleReproducible(t *testing.T) {
	a, b := SandboxWebhookSample(), SandboxWebhookSample()
	if a != b {
		t.Fatal("webhook sample must be deterministic for replay")
	}
	// 复算 v1 与样例一致(与 scripts/openplat-selftest.mjs 同算法)。
	v1 := SignPayload("ops_sandbox_demo", a.Timestamp, []byte(a.Body))
	if v1 != a.ExpectedV1 {
		t.Fatalf("expectedV1 mismatch: %s vs %s", v1, a.ExpectedV1)
	}
	if a.EventType != "order.stage.done" || a.EventID == "" {
		t.Fatalf("unexpected sample: %+v", a)
	}
}
