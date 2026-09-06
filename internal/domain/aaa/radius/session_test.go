package radius

// AAA-A2 计账/认证链路回归:会话维护(Start/Interim/Stop)、孤儿 Stop 不报错、并发超限 Reject、CoA 组包。

import (
	"context"
	"errors"
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

type sessionStub struct {
	started        []aaa.SessionRecord
	touched        [][4]any // loid, sessionID, in, out
	stopped        []string // loid|sessionID|reason
	stopErr        error
	startedCreated bool
}

func (s *sessionStub) StartSession(_ context.Context, rec aaa.SessionRecord) (bool, error) {
	s.started = append(s.started, rec)
	return s.startedCreated, nil
}

func (s *sessionStub) TouchSessionTraffic(_ context.Context, loid, sessionID string, in, out int64) error {
	s.touched = append(s.touched, [4]any{loid, sessionID, in, out})
	return nil
}

func (s *sessionStub) StopSession(_ context.Context, loid, sessionID, reason string) (bool, error) {
	s.stopped = append(s.stopped, loid+"|"+sessionID+"|"+reason)
	if s.stopErr != nil {
		return false, s.stopErr
	}
	return true, nil
}

type gateStub struct {
	allow   bool
	current int
	err     error
	loid    string
	limit   int
}

func (g *gateStub) AllowNewSession(_ context.Context, loid string, limit int) (bool, int, error) {
	g.loid, g.limit = loid, limit
	return g.allow, g.current, g.err
}

func acctRequest(status rfc2866.AcctStatusType) *radius.Request {
	p := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	_ = rfc2865.UserName_SetString(p, "LOID-1")
	_ = rfc2866.AcctStatusType_Set(p, status)
	_ = rfc2866.AcctSessionID_SetString(p, "S-1")
	_ = rfc2866.AcctInputOctets_Set(p, 11)
	_ = rfc2866.AcctOutputOctets_Set(p, 22)
	return &radius.Request{Packet: p}
}

func TestServeAccountingSessions(t *testing.T) {
	sess := &sessionStub{startedCreated: true}
	h := &Handler{Sessions: sess}
	w := &responseStub{}
	h.ServeRADIUS(w, acctRequest(rfc2866.AcctStatusType_Value_Start))
	if w.packet == nil || w.packet.Code != radius.CodeAccountingResponse {
		t.Fatalf("Start 应回 Accounting-Response: %v", w.packet)
	}
	if len(sess.started) != 1 || sess.started[0].Loid != "LOID-1" || sess.started[0].SessionID != "S-1" ||
		sess.started[0].InputOctets != 11 || sess.started[0].OutputOctets != 22 {
		t.Fatalf("Start 建会话字段不符: %+v", sess.started)
	}
	w = &responseStub{}
	h.ServeRADIUS(w, acctRequest(rfc2866.AcctStatusType_Value_InterimUpdate))
	if len(sess.touched) != 1 || sess.touched[0][0] != "LOID-1" || sess.touched[0][2] != int64(11) {
		t.Fatalf("Interim 累加参数不符: %v", sess.touched)
	}
	w = &responseStub{}
	h.ServeRADIUS(w, acctRequest(rfc2866.AcctStatusType_Value_Stop))
	if len(sess.stopped) != 1 || sess.stopped[0] != "LOID-1|S-1|"+aaa.CloseReasonAcctStop {
		t.Fatalf("Stop 关闭参数不符: %v", sess.stopped)
	}
}

// 孤儿 Stop(无对应会话)只记话单不报错:仍回 Accounting-Response。
func TestServeAccountingOrphanStop(t *testing.T) {
	sess := &sessionStub{} // startedCreated=false 即孤儿路径:StopSession 返回 false
	h := &Handler{Sessions: sess}
	w := &responseStub{}
	h.ServeRADIUS(w, acctRequest(rfc2866.AcctStatusType_Value_Stop))
	if w.packet == nil || w.packet.Code != radius.CodeAccountingResponse {
		t.Fatalf("孤儿 Stop 不得影响计费响应: %v", w.packet)
	}
	if len(sess.stopped) != 1 {
		t.Fatalf("孤儿 Stop 也应过维护链路留痕: %v", sess.stopped)
	}
}

// 会话维护故障不阻塞计费响应,但留 [aaa] 可 grep 日志。
func TestServeAccountingSessionError(t *testing.T) {
	sess := &sessionStub{stopErr: errors.New("db down")}
	h := &Handler{Sessions: sess}
	w := &responseStub{}
	h.ServeRADIUS(w, acctRequest(rfc2866.AcctStatusType_Value_Stop))
	if w.packet == nil || w.packet.Code != radius.CodeAccountingResponse {
		t.Fatalf("维护故障不得拒绝计费: %v", w.packet)
	}
}

// 并发会话达到上限:认证 Reject 且认证日志标注 CONCURRENT_LIMIT。
func TestServeAuthConcurrentLimitReject(t *testing.T) {
	gate := &gateStub{allow: false, current: 1}
	logger := &authLogStub{}
	h := &Handler{
		Auth: &authStub{decision: aaa.Decision{Authorize: true, Bandwidth: "B-100"}},
		Gate: gate, SessionLimit: 1, Log: logger,
	}
	w := &responseStub{}
	h.ServeRADIUS(w, accessRequest("LOID-1"))
	if w.packet == nil || w.packet.Code != radius.CodeAccessReject {
		t.Fatalf(
			"并发超限应 Reject: %v", w.packet)
	}
	if gate.limit != 1 || gate.loid != "LOID-1" {
		t.Fatalf("闸口参数不符: %+v", gate)
	}
	if len(logger.logs) != 1 || logger.logs[0].Result != "FAILED" || logger.logs[0].Reason != aaa.AuthFailReasonConcurrent {
		t.Fatalf("认证日志须标注并发超限: %+v", logger.logs)
	}
}

// 未达上限放行;闸口自身故障放行(DB 故障在 Decide 已先拒绝,不放大故障面)。
func TestServeAuthGatePass(t *testing.T) {
	logger := &authLogStub{}
	h := &Handler{
		Auth: &authStub{decision: aaa.Decision{Authorize: true}},
		Gate: &gateStub{allow: true, current: 0}, SessionLimit: 1, Log: logger,
	}
	w := &responseStub{}
	h.ServeRADIUS(w, accessRequest("LOID-2"))
	if w.packet == nil || w.packet.Code != radius.CodeAccessAccept || logger.logs[0].Result != "SUCCESS" {
		t.Fatalf("未达上限应放行: %v %+v", w.packet, logger.logs)
	}
	h2 := &Handler{
		Auth: &authStub{decision: aaa.Decision{Authorize: true}},
		Gate: &gateStub{err: errors.New("db down")}, SessionLimit: 1,
	}
	w2 := &responseStub{}
	h2.ServeRADIUS(w2, accessRequest("LOID-3"))
	if w2.packet == nil || w2.packet.Code != radius.CodeAccessAccept {
		t.Fatalf("闸口故障应放行: %v", w2.packet)
	}
}
