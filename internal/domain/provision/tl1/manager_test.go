package tl1

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// fastBackoff 缩短退避等待,返回还原函数。
func fastBackoff() func() {
	old := retryBackoff
	retryBackoff = 5 * time.Millisecond
	return func() { retryBackoff = old }
}

func TestManagerLazyDial(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{
		{data: compldFrame("1")},
		{data: compldFrame("B000001")},
	}}
	restore := stubDialNet(fc)
	defer restore()
	m := NewManager(Config{Addr: "192.0.2.1:13027", User: "u", Pass: "p"})
	resp, err := m.Do(context.Background(), Command{Verb: "LST-ONU"})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.Completion != "COMPLD" {
		t.Fatalf("resp=%+v", resp)
	}
	if !strings.Contains(fc.sent(), "LOGIN:::1::UN=u,PWD=p;") {
		t.Fatalf("sent %q", fc.sent())
	}
}

func TestManagerReconnectOnce(t *testing.T) {
	defer fastBackoff()()
	broken := &fakeConn{steps: []fakeStep{
		{data: compldFrame("1")},
		{err: io.EOF},
	}}
	good := &fakeConn{steps: []fakeStep{
		{data: compldFrame("1")},
		{data: compldFrame("B000001")},
	}}
	restore := stubDialNet(broken, good)
	defer restore()
	m := NewManager(Config{Addr: "192.0.2.1:13027", User: "u", Pass: "p"})
	resp, err := m.Do(context.Background(), Command{Verb: "LST-ONU"})
	if err != nil {
		t.Fatalf("Do after reconnect: %v", err)
	}
	if resp.Completion != "COMPLD" {
		t.Fatalf("resp=%+v", resp)
	}
	if !strings.Contains(good.sent(), "LOGIN") {
		t.Fatalf("expected redial, good.sent=%q", good.sent())
	}
}

func TestManagerAuthBreaker(t *testing.T) {
	deny := &fakeConn{steps: []fakeStep{{data: denyFrame("1")}}}
	next := &fakeConn{steps: []fakeStep{
		{data: compldFrame("1")},
		{data: compldFrame("B000001")},
	}}
	restore := stubDialNet(deny, next)
	defer restore()
	m := NewManager(Config{Addr: "192.0.2.1:13027", User: "bad", Pass: "bad"})
	if _, err := m.Do(context.Background(), Command{Verb: "LST-ONU"}); !errors.Is(err, ErrAuth) {
		t.Fatalf("want ErrAuth got %v", err)
	}
	// 熔断窗口内不再建连
	if _, err := m.Do(context.Background(), Command{Verb: "LST-ONU"}); !errors.Is(err, ErrAuth) {
		t.Fatalf("breaker: want ErrAuth got %v", err)
	}
	if strings.Contains(next.sent(), "LOGIN") {
		t.Fatalf("breaker window must not redial, sent=%q", next.sent())
	}
	// 窗口过后恢复建连
	m.mu.Lock()
	m.authFailAt = time.Now().Add(-breakerWindow - time.Second)
	m.mu.Unlock()
	resp, err := m.Do(context.Background(), Command{Verb: "LST-ONU"})
	if err != nil {
		t.Fatalf("after breaker: %v", err)
	}
	if resp.Completion != "COMPLD" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestWithSessionRerunOnBroken(t *testing.T) {
	defer fastBackoff()()
	broken := &fakeConn{steps: []fakeStep{
		{data: compldFrame("1")},
		{err: io.EOF},
	}}
	good := &fakeConn{steps: []fakeStep{
		{data: compldFrame("1")},
		{data: compldFrame("B000001")},
		{data: compldFrame("B000002")},
	}}
	restore := stubDialNet(broken, good)
	defer restore()
	m := NewManager(Config{Addr: "192.0.2.1:13027", User: "u", Pass: "p"})
	calls := 0
	err := m.WithSession(context.Background(), func(s *Session) error {
		calls++
		for i := 0; i < 2; i++ {
			if _, err := s.Do(context.Background(), Command{Verb: "LST-ONU"}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithSession: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d want 2(断线整段重跑)", calls)
	}
}

func TestManagerUseResetsEndpointAndBreaker(t *testing.T) {
	m := NewManager(Config{Addr: "old.example:13027", User: "old", Pass: "old-pass"})
	m.authFailAt = time.Now()
	m.Use(Endpoint{Host: "new.example", Port: 14027, User: "new", Pass: "new-pass"})
	if got := m.Endpoint(); got != (Endpoint{Host: "new.example", Port: 14027, User: "new", Pass: "new-pass"}) {
		t.Fatalf("endpoint=%+v", got)
	}
	if m.inBreaker() {
		t.Fatal("Use must clear old auth breaker")
	}
}
