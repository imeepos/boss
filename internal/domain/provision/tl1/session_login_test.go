package tl1

import (
	"context"
	"errors"
	"testing"
)

// login 的 DELAY 追帧回归:真实 U2000 对 LOGIN 也可能先回 DELAY(PDF §11 未承诺 LOGIN 免延迟),
// 修复前 login 只读一帧即判 COMPLD/DENY,收到 DELAY 直接落 ErrAuth(ISSUE 2026-09-02)。

func TestDialLoginDelayThenCompld(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{
		{data: delayFrame("1")},
		{data: compldFrame("1")},
	}}
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

func TestDialLoginDelayThenDeny(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{
		{data: delayFrame("1")},
		{data: denyFrame("1")},
	}}
	restore := stubDialNet(fc)
	defer restore()
	_, err := Dial(context.Background(), Config{Addr: "192.0.2.1:13027", User: "bad", Pass: "bad"})
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("want ErrAuth got %v", err)
	}
}

func TestDialLoginStrayFrameThenCompld(t *testing.T) {
	// 异 ctag 残帧(如网管主动上报)应丢弃,继续等同 ctag 最终帧。
	fc := &fakeConn{steps: []fakeStep{
		{data: compldFrame("B000042")},
		{data: compldFrame("1")},
	}}
	restore := stubDialNet(fc)
	defer restore()
	s, err := Dial(context.Background(), Config{Addr: "192.0.2.1:13027", User: "tester", Pass: "Changeme_321"})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer s.Close()
}
