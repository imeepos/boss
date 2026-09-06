package radius

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"log"
	"net"
	"os"
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
	creds    aaa.Credentials
}

func (s *authStub) Authenticate(_ context.Context, loid string, creds aaa.Credentials) (aaa.Decision, error) {
	s.loid = loid
	s.creds = creds
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
	err  error
}

func (s *authLogStub) AppendAuthLog(_ context.Context, l aaa.AuthLog) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
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
	if len(logger.logs) != 1 || logger.logs[0].Loid != "LOID-1" || logger.logs[0].Result != "SUCCESS" || logger.logs[0].FailReason != "" {
		t.Fatalf("auth log: %+v", logger.logs)
	}

	// 拒绝场景也写日志
	logger2 := &authLogStub{}
	h2 := &Handler{Auth: &authStub{err: aaa.ErrNotFound}, Log: logger2}
	h2.ServeRADIUS(&responseStub{}, accessRequest("LOID-2"))
	if len(logger2.logs) != 1 || logger2.logs[0].Loid != "LOID-2" || logger2.logs[0].Result != "FAILED" || logger2.logs[0].FailReason != aaa.FailReasonNotFound {
		t.Fatalf("reject auth log: %+v", logger2.logs)
	}
}

// TestServeAuthParseCredentials A1:凭据属性提取(PAP 明文 / CHAP ident+challenge+摘要)。
func TestServeAuthParseCredentials(t *testing.T) {
	t.Run("PAP", func(t *testing.T) {
		a := &authStub{decision: aaa.Decision{Authorize: true}}
		h := &Handler{Auth: a}
		r := accessRequest("LOID-PA")
		_ = rfc2865.UserPassword_SetString(r.Packet, "pap-secret")
		h.ServeRADIUS(&responseStub{}, r)
		if a.creds.PAP != "pap-secret" || a.creds.CHAP != nil {
			t.Fatalf("creds=%+v", a.creds)
		}
	})

	t.Run("CHAP 带 Challenge", func(t *testing.T) {
		a := &authStub{decision: aaa.Decision{Authorize: true}}
		h := &Handler{Auth: a}
		r := accessRequest("LOID-CH")
		sum := md5.Sum([]byte("anything"))
		_ = rfc2865.CHAPPassword_Set(r.Packet, append([]byte{9}, sum[:]...))
		challenge := []byte{0xAA, 0xBB, 0xCC}
		_ = rfc2865.CHAPChallenge_Set(r.Packet, challenge)
		h.ServeRADIUS(&responseStub{}, r)
		if a.creds.CHAP == nil || a.creds.CHAP.Ident != 9 || string(a.creds.CHAP.Challenge) != string(challenge) || len(a.creds.CHAP.Response) != 16 {
			t.Fatalf("creds=%+v", a.creds.CHAP)
		}
	})

	t.Run("CHAP 无 Challenge 缺省 Request Authenticator", func(t *testing.T) {
		a := &authStub{decision: aaa.Decision{Authorize: true}}
		h := &Handler{Auth: a}
		r := accessRequest("LOID-CN")
		sum := md5.Sum([]byte("x"))
		_ = rfc2865.CHAPPassword_Set(r.Packet, append([]byte{1}, sum[:]...))
		h.ServeRADIUS(&responseStub{}, r)
		if a.creds.CHAP == nil || string(a.creds.CHAP.Challenge) != string(r.Packet.Authenticator[:]) {
			t.Fatalf("challenge=%v want authenticator", a.creds.CHAP)
		}
	})
}

// TestServeAuthFailReasonLogged A1:失败原因码写入认证日志(fields.md §8A 枚举)。
func TestServeAuthFailReasonLogged(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{aaa.ErrLocked, aaa.FailReasonLocked},
		{aaa.ErrNotFound, aaa.FailReasonNotFound},
		{aaa.ErrSuspended, aaa.FailReasonSuspended},
		{aaa.ErrClosed, aaa.FailReasonClosed},
		{aaa.ErrBadCredential, aaa.FailReasonBadCredential},
		{errors.New("db down"), aaa.FailReasonBadCredential},
	}
	for _, tc := range cases {
		logger := &authLogStub{}
		h := &Handler{Auth: &authStub{err: tc.err}, Log: logger}
		h.ServeRADIUS(&responseStub{}, accessRequest("LOID-R"))
		if len(logger.logs) != 1 || logger.logs[0].FailReason != tc.want {
			t.Fatalf("err=%v logs=%+v want %s", tc.err, logger.logs, tc.want)
		}
	}
}

// TestServeAuthLogWriteFailureObservable A1 留痕红线:认证日志写失败不得静默,
// 必须输出带 [aaa] 前缀与 FAILED 的可 grep 日志并附 LOID 上下文。
func TestServeAuthLogWriteFailureObservable(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	logger := &authLogStub{err: errors.New("pg connection refused")}
	h := &Handler{Auth: &authStub{decision: aaa.Decision{Authorize: true}}, Log: logger}
	h.ServeRADIUS(&responseStub{}, accessRequest("LOID-9X"))
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("[aaa]")) || !bytes.Contains([]byte(out), []byte("FAILED")) || !bytes.Contains([]byte(out), []byte("LOID-9X")) {
		t.Fatalf("log output missing red line markers: %q", out)
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
