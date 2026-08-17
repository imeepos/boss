package billing

import (
	"context"
	"testing"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// PGEmitter 话单落库单测:字段映射 + UNBILLED 状态。
func TestPGEmitter_Emit(t *testing.T) {
	svc := &fakeAaaWriter{}
	e := NewPGEmitter(svc)
	if err := e.Emit(context.Background(), CDR{
		LOID: "LOID-1", Username: "u1", AcctStatus: 2, SessionID: "S-1",
		SessionTime: 3600, InputOctets: 1024, OutputOctets: 2048, NASIP: "10.0.0.1",
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	c := svc.captured
	if c.Loid != "LOID-1" || c.AcctStatus != 2 || c.SessionTime != 3600 ||
		c.InputOctets != 1024 || c.OutputOctets != 2048 || c.NasIP != "10.0.0.1" {
		t.Fatalf("captured=%+v", c)
	}
	if c.BillingStatus != "UNBILLED" {
		t.Fatalf("billingStatus=%s", c.BillingStatus)
	}
}

type fakeAaaWriter struct{ captured aaa.CdrRecord }

func (f *fakeAaaWriter) AppendCdr(_ context.Context, c aaa.CdrRecord) (int64, error) {
	f.captured = c
	return 1, nil
}
