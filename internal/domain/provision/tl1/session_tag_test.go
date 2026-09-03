package tl1

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// ---- 业务 ctag(Tag 位)与 Session.Do 响应匹配 ----

// TestSessionDoBusinessTag 业务 Tag 写线进 ctag 位;响应按实际 ctag 匹配,
// 异 ctag 残帧被丢弃,自增 B 位不出现在业务命令上。
func TestSessionDoBusinessTag(t *testing.T) {
	fc := &fakeConn{steps: []fakeStep{
		{data: compldFrame("B000002")}, // 异 ctag 残帧应被丢弃
		{data: compldFrame("ADDONT")},
	}}
	s := newRawSession(fc, time.Minute)
	resp, err := s.Do(context.Background(), Command{Verb: "ADD-ONU", Tag: "ADDONT", Payload: []KV{{"ONUID", "loid-0001"}}})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.CTag != "ADDONT" || resp.Completion != "COMPLD" {
		t.Fatalf("resp=%+v", resp)
	}
	if got, want := fc.sent(), "ADD-ONU:::ADDONT::ONUID=loid-0001;"; !strings.Contains(got, want) {
		t.Fatalf("sent %q want contain %q", got, want)
	}
	if strings.Contains(fc.sent(), ":B00000") {
		t.Fatalf("business tag command must not carry auto ctag: %q", fc.sent())
	}
	s.Close()
}

// TestSessionDoBadTag 白名单外 Tag 在 Do 构建期即拒,不产生任何写线。
func TestSessionDoBadTag(t *testing.T) {
	fc := &fakeConn{}
	s := newRawSession(fc, time.Minute)
	defer s.Close()
	_, err := s.Do(context.Background(), Command{Verb: "ADD-ONU", Tag: "A:B"})
	if !errors.Is(err, ErrBadParam) {
		t.Fatalf("want ErrBadParam got %v", err)
	}
	if fc.sent() != "" {
		t.Fatalf("nothing should hit wire, sent %q", fc.sent())
	}
}
