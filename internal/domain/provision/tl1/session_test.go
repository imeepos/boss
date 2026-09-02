package tl1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- fake conn 桩(仅测试,不进生产文件) ----

// fakeStep 读脚本单步:先逐字节吐 data,再返回 err;二者可任一为空。
type fakeStep struct {
	data string
	err  error
}

// fakeConn 可控 net.Conn:写入全量留痕供断言,读取按脚本推进。
type fakeConn struct {
	mu       sync.Mutex
	writes   []byte
	steps    []fakeStep
	idx, pos int
	closed   bool
	deadline time.Time
}

func (f *fakeConn) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, io.EOF
	}
	for f.idx < len(f.steps) {
		st := f.steps[f.idx]
		if f.pos < len(st.data) {
			p[0] = st.data[f.pos]
			f.pos++
			return 1, nil
		}
		f.idx++
		f.pos = 0
		if st.err != nil {
			return 0, st.err
		}
	}
	if !f.deadline.IsZero() && time.Now().After(f.deadline) {
		return 0, os.ErrDeadlineExceeded
	}
	return 0, io.EOF
}

func (f *fakeConn) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, errors.New("fake conn closed")
	}
	f.writes = append(f.writes, p...)
	return len(p), nil
}

func (f *fakeConn) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr              { return fakeAddr{} }
func (f *fakeConn) RemoteAddr() net.Addr             { return fakeAddr{} }
func (f *fakeConn) SetDeadline(time.Time) error      { return nil }
func (f *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	f.mu.Lock()
	f.deadline = t
	f.mu.Unlock()
	return nil
}

// sent 返回会话已写入的全部指令文本。
func (f *fakeConn) sent() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return string(f.writes)
}

type fakeAddr struct{}

func (fakeAddr) Network() string { return "fake" }
func (fakeAddr) String() string  { return "fake" }

// stubDialNet 让 Dial 按序返回给定 fake conn;返回还原函数。
func stubDialNet(conns ...*fakeConn) func() {
	old := dialNet
	i := 0
	dialNet = func(context.Context, string, string, time.Duration) (net.Conn, error) {
		if i >= len(conns) {
			return nil, errors.New("no more fake conns scripted")
		}
		c := conns[i]
		i++
		return c, nil
	}
	return func() { dialNet = old }
}

// ---- 响应帧样例(逐字段对齐 PDF §10.3) ----

func opFrame(ctag, compl string, en int, desc string) string {
	return fmt.Sprintf("   HW_10.71.227.225 2026-09-02 10:00:00\nM  %s %s\n   EN=%d   ENDESC=%s\n;", ctag, compl, en, desc)
}

func compldFrame(ctag string) string { return opFrame(ctag, "COMPLD", 0, "成功。") }
func delayFrame(ctag string) string  { return opFrame(ctag, "DELAY", 0, "等待。") }
func denyFrame(ctag string) string {
	return opFrame(ctag, "DENY", 76546031, "用户名或密码错误。")
}

// newRawSession 绕过 LOGIN 构造会话,供 Do/SHAKEHAND 定向测试。
func newRawSession(fc *fakeConn, keepalive time.Duration) *Session {
	cfg := Config{}.norm()
	cfg.Keepalive = keepalive
	return &Session{conn: fc, cfg: cfg, stopCh: make(chan struct{})}
}

// ---- Dial ----

func TestDialLoginOK(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{{data: compldFrame("1")}}}
	restore := stubDialNet(fc)
	defer restore()
	s, err := Dial(context.Background(), Config{Addr: "192.0.2.1:13027", User: "tester", Pass: "Changeme_321"})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer s.Close()
	if got, want := fc.sent(), "LOGIN:::1::UN=tester,PWD=Changeme_321;"; got != want {
		t.Fatalf("sent %q want %q", got, want)
	}
}

func TestDialAuthDenied(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{{data: denyFrame("1")}}}
	restore := stubDialNet(fc)
	defer restore()
	_, err := Dial(context.Background(), Config{Addr: "192.0.2.1:13027", User: "bad", Pass: "bad"})
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("want ErrAuth got %v", err)
	}
	if !strings.Contains(fc.sent(), "UN=bad") {
		t.Fatalf("sent %q", fc.sent())
	}
}

// ---- Do ----

func TestSessionDoCompld(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{{data: compldFrame("B000001")}}}
	s := newRawSession(fc, time.Minute)
	defer s.Close()
	resp, err := s.Do(context.Background(), Command{Verb: "LST-ONU", Payload: []KV{{"ONUID", "1"}}})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.CTag != "B000001" || resp.Completion != "COMPLD" || resp.EN != 0 {
		t.Fatalf("resp=%+v", resp)
	}
	if got, want := fc.sent(), "LST-ONU:::B000001::ONUID=1;"; got != want {
		t.Fatalf("sent %q want %q", got, want)
	}
}

func TestSessionDoDelayThenFinal(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{
		{data: delayFrame("B000001")},
		{data: compldFrame("B000001")},
	}}
	s := newRawSession(fc, time.Minute)
	defer s.Close()
	resp, err := s.Do(context.Background(), Command{Verb: "ADD-ONU"})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.Completion != "COMPLD" || resp.CTag != "B000001" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestSessionDoDenyPassthrough(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{{data: denyFrame("B000001")}}}
	s := newRawSession(fc, time.Minute)
	defer s.Close()
	resp, err := s.Do(context.Background(), Command{Verb: "ADD-ONU"})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.Completion != "DENY" || resp.EN != 76546031 {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestSessionDoReadTimeout(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{
		{data: delayFrame("B000001")},
		{err: os.ErrDeadlineExceeded},
	}}
	s := newRawSession(fc, time.Minute)
	defer s.Close()
	_, err := s.Do(context.Background(), Command{Verb: "ADD-ONU"})
	if !errors.Is(err, ErrConnBroken) {
		t.Fatalf("want ErrConnBroken got %v", err)
	}
}

func TestSessionDoEOF(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{{err: io.EOF}}}
	s := newRawSession(fc, time.Minute)
	defer s.Close()
	_, err := s.Do(context.Background(), Command{Verb: "LST-ONU"})
	if !errors.Is(err, ErrConnBroken) {
		t.Fatalf("want ErrConnBroken got %v", err)
	}
}

// ---- SHAKEHAND / Close ----

func TestSessionShakehandOnIdle(t *testing.T) {
	fc := &fakeConn{}
	s := newRawSession(fc, 20*time.Millisecond)
	s.startKeepalive()
	time.Sleep(120 * time.Millisecond)
	s.Close()
	if got := fc.sent(); !strings.Contains(got, "SHAKEHAND:::B000001::;") {
		t.Fatalf("sent %q", got)
	}
}

func TestSessionCloseIdempotent(t *testing.T) {
	fc := &fakeConn{}
	s := newRawSession(fc, time.Minute)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if !strings.Contains(fc.sent(), "LOGOUT:::B000001::;") {
		t.Fatalf("sent %q", fc.sent())
	}
}

func TestSessionDoAfterClose(t *testing.T) {
	s := newRawSession(&fakeConn{}, time.Minute)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := s.Do(context.Background(), Command{Verb: "LST-ONU"})
	if !errors.Is(err, ErrConnBroken) {
		t.Fatalf("want ErrConnBroken got %v", err)
	}
}
