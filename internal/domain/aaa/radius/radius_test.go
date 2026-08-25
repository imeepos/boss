package radius

import (
	"context"
	"net"
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/ymm-001/boss/internal/domain/aaa"
	billing "github.com/ymm-001/boss/internal/domain/aaa/billing"
)

type authStub struct {
	decision aaa.Decision
	err      error
	loid     string
}

func (s *authStub) Decide(_ context.Context, loid string) (aaa.Decision, error) {
	s.loid = loid
	return s.decision, s.err
}

type emitterStub struct {
	cdr billing.CDR
	err error
}

func (s *emitterStub) Emit(_ context.Context, cdr billing.CDR) error {
	s.cdr = cdr
	return s.err
}

type responseStub struct{ packet *radius.Packet }

func (s *responseStub) Write(p *radius.Packet) error {
	s.packet = p
	return nil
}

type authLogStub struct {
	logs []aaa.AuthLog
}

func (s *authLogStub) AppendAuthLog(_ context.Context, l aaa.AuthLog) (int64, error) {
	s.logs = append(s.logs, l)
	return int64(len(s.logs)), nil
}

func accessRequest(loid string) *radius.Request {
	p := radius.New(radius.CodeAccessRequest, []byte("secret"))
	if loid != "" {
		_ = rfc2865.UserName_SetString(p, loid)
	}
	return &radius.Request{Packet: p}
}

func TestServeAuthBranches(t *testing.T) {
	tests := []struct {
		name     string
		loid     string
		stub     authStub
		code     radius.Code
		wantLOID string
	}{
		{name: "missing user", code: radius.CodeAccessReject},
		{name: "decision error", loid: "L2", stub: authStub{err: aaa.ErrNotFound}, code: radius.CodeAccessReject, wantLOID: "L2"},
		{name: "denied", loid: "L3", stub: authStub{decision: aaa.Decision{Authorize: false}}, code: radius.CodeAccessReject, wantLOID: "L3"},
		{name: "accepted", loid: "L4", stub: authStub{decision: aaa.Decision{Authorize: true, Bandwidth: "B-100", SessionTTL: 60}}, code: radius.CodeAccessAccept, wantLOID: "L4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &authStub{decision: tt.stub.decision, err: tt.stub.err}
			h := &Handler{Auth: a}
			w := &responseStub{}
			h.ServeRADIUS(w, accessRequest(tt.loid))
			if w.packet == nil || w.packet.Code != tt.code {
				t.Fatalf("response=%v want %v", w.packet, tt.code)
			}
			if a.loid != tt.wantLOID {
				t.Fatalf("loid=%q want %q", a.loid, tt.wantLOID)
			}
		})
	}
}

func TestServeAuthWritesLog(t *testing.T) {
	logger := &authLogStub{}
	h := &Handler{Auth: &authStub{decision: aaa.Decision{Authorize: true, Bandwidth: "B-100"}}, Log: logger}
	w := &responseStub{}
	h.ServeRADIUS(w, accessRequest("LOID-1"))
	if w.packet == nil || w.packet.Code != radius.CodeAccessAccept {
		t.Fatalf("accept failed: %v", w.packet)
	}
	if len(logger.logs) != 1 || logger.logs[0].Loid != "LOID-1" || logger.logs[0].Result != "SUCCESS" {
		t.Fatalf("auth log: %+v", logger.logs)
	}

	// 拒绝场景也写日志
	logger2 := &authLogStub{}
	h2 := &Handler{Auth: &authStub{err: aaa.ErrNotFound}, Log: logger2}
	h2.ServeRADIUS(&responseStub{}, accessRequest("LOID-2"))
	if len(logger2.logs) != 1 || logger2.logs[0].Loid != "LOID-2" || logger2.logs[0].Result != "FAILED" {
		t.Fatalf("reject auth log: %+v", logger2.logs)
	}
}

func TestServeRADIUSUnknownCode(t *testing.T) {
	w := &responseStub{}
	p := radius.New(radius.Code(99), []byte("secret"))
	(&Handler{}).ServeRADIUS(w, &radius.Request{Packet: p})
	if w.packet != nil {
		t.Fatalf("unknown code wrote response: %v", w.packet.Code)
	}
}

func TestServeAccounting(t *testing.T) {
	p := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	_ = rfc2865.UserName_SetString(p, "LOID-1")
	_ = rfc2866.AcctStatusType_Set(p, rfc2866.AcctStatusType_Value_Start)
	_ = rfc2866.AcctSessionID_SetString(p, "session-1")
	_ = rfc2866.AcctInputOctets_Set(p, 11)
	_ = rfc2866.AcctOutputOctets_Set(p, 22)
	_ = rfc2865.NASIPAddress_Set(p, net.IPv4(192, 0, 2, 1))
	e := &emitterStub{}
	w := &responseStub{}
	(&Handler{CDR: e}).ServeRADIUS(w, &radius.Request{Packet: p})
	if w.packet == nil || w.packet.Code != radius.CodeAccountingResponse {
		t.Fatalf("response=%v", w.packet)
	}
	if e.cdr.LOID != "LOID-1" || e.cdr.SessionID != "session-1" || e.cdr.InputOctets != 11 || e.cdr.OutputOctets != 22 {
		t.Fatalf("cdr=%+v", e.cdr)
	}
	w = &responseStub{}
	(&Handler{}).ServeRADIUS(w, &radius.Request{Packet: p})
	if w.packet == nil {
		t.Fatal("nil emitter should still respond")
	}
}

func TestNewServer(t *testing.T) {
	s := New(":0", []byte("secret"), &Handler{})
	if s == nil || s.srv == nil || s.done == nil {
		t.Fatalf("server=%+v", s)
	}
}

func TestServerStartInvalidAddress(t *testing.T) {
	if err := New("bad-address", []byte("secret"), &Handler{}).Start(); err == nil {
		t.Fatal("Start should reject invalid address")
	}
}

func TestServerShutdownBeforeStart(t *testing.T) {
	if err := New(":0", []byte("secret"), &Handler{}).Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
