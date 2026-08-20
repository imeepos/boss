package sms

import (
	"context"
	"testing"
)

func TestLogSender(t *testing.T) {
	s := NewLogSender()
	if _, ok := s.(LogSender); !ok {
		t.Fatalf("sender=%T, want LogSender", s)
	}
	if err := s.Send(context.Background(), "+8613800138000", "123456", "login"); err != nil {
		t.Fatalf("Send: %v", err)
	}
}
