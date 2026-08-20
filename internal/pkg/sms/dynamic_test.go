package sms

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewDynamic_LogFallback(t *testing.T) {
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		return ChannelConfig{Enabled: true}, nil // 凭据为空 → LogSender
	}).(*Dynamic)
	if err := d.Send(context.Background(), "+8613800138000", "123", "login"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, ok := d.cached.sender.(LogSender); !ok {
		t.Fatalf("cached=%T, want LogSender", d.cached.sender)
	}
}

func TestDynamic_AliyunPath(t *testing.T) {
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		return ChannelConfig{
			Enabled: true, AccessKeyID: "ak", AccessKeySecret: "sk", From: "F",
			TemplateCN: "cn{code}", TemplateMY: "my{code}",
		}, nil
	}).(*Dynamic)
	s, err := d.current(context.Background())
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if _, ok := s.(*aliyunIntl); !ok {
		t.Fatalf("sender=%T, want *aliyunIntl", s)
	}
}

func TestDynamic_Disabled(t *testing.T) {
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		return ChannelConfig{Enabled: false}, nil
	})
	if err := d.Send(context.Background(), "+8613800138000", "1", "login"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("err=%v, want ErrDisabled", err)
	}
}

func TestDynamic_ResolveError(t *testing.T) {
	want := errors.New("db fail")
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		return ChannelConfig{}, want
	})
	if err := d.Send(context.Background(), "+8613800138000", "1", "login"); !errors.Is(err, want) {
		t.Fatalf("err=%v, want %v", err, want)
	}
}

func TestDynamic_CacheTTL(t *testing.T) {
	calls := 0
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		calls++
		return ChannelConfig{Enabled: true}, nil
	}).(*Dynamic)
	ctx := context.Background()
	// 未命中缓存 → resolve。
	if _, err := d.current(ctx); err != nil {
		t.Fatal(err)
	}
	// TTL 内命中缓存 → 不 resolve。
	if _, err := d.current(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("resolve calls=%d, want 1", calls)
	}
	// 过期 → 重新 resolve。
	d.mu.Lock()
	d.cached.at = time.Now().Add(-2 * d.ttl)
	d.mu.Unlock()
	if _, err := d.current(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("resolve calls=%d, want 2", calls)
	}
}
