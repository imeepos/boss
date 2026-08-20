package sms

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeE164(t *testing.T) {
	cases := []struct{ in, want string }{
		{"+8613800138000", "+8613800138000"},
		{"8613800138000", "+8613800138000"},
		{"13800138000", "+8613800138000"},
		{"+60123456789", "+60123456789"},
		{"+60 12-345 6789", "+60123456789"},
	}
	for _, c := range cases {
		got, err := NormalizeE164(c.in)
		if err != nil || got != c.want {
			t.Errorf("NormalizeE164(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"+19995551234", "+86", "", "abc"} {
		if _, err := NormalizeE164(bad); !errors.Is(err, ErrUnsupportedRegion) {
			t.Errorf("NormalizeE164(%q) err = %v; want ErrUnsupportedRegion", bad, err)
		}
	}
}

type fakeSender struct{ got, region string }

func (f *fakeSender) Send(_ context.Context, phone, code, _ string) error {
	f.got = phone + ":" + code
	f.region = Region(phone)
	return nil
}

func TestRouterRegionRouting(t *testing.T) {
	def := &fakeSender{}
	my := &fakeSender{}
	r := NewRouter(def, map[string]Sender{"60": my}, []string{"86", "60"})
	if err := r.Send(context.Background(), "13800138000", "123456", "login"); err != nil {
		t.Fatal(err)
	}
	if def.region != "86" {
		t.Errorf("CN routed to %q; want default(86)", def.region)
	}
	if err := r.Send(context.Background(), "+60123456789", "654321", "login"); err != nil {
		t.Fatal(err)
	}
	if my.region != "60" {
		t.Errorf("MY routed to %q; want 60 override", my.region)
	}
	if err := r.Send(context.Background(), "+19995551234", "111111", "login"); !errors.Is(err, ErrUnsupportedRegion) {
		t.Errorf("US err = %v; want ErrUnsupportedRegion", err)
	}
}

func TestRouterUnsupportedAndNoDefault(t *testing.T) {
	def := &fakeSender{}
	// 区号可归一化但不在 supported 列表。
	r := NewRouter(def, nil, []string{"86"})
	err := r.Send(context.Background(), "+60123456789", "1", "login")
	if !errors.Is(err, ErrUnsupportedRegion) {
		t.Fatalf("err=%v, want ErrUnsupportedRegion", err)
	}
	// ByRegion 命中但 sender 为 nil → 走 Default。
	r = NewRouter(def, map[string]Sender{"86": nil}, []string{"86"})
	if err := r.Send(context.Background(), "13800138000", "1", "login"); err != nil {
		t.Fatal(err)
	}
	if def.region != "86" {
		t.Fatalf("routed=%q, want default", def.region)
	}
	// Default 为 nil 且无 ByRegion。
	r = NewRouter(nil, nil, []string{"86"})
	if err := r.Send(context.Background(), "13800138000", "1", "login"); err == nil ||
		err.Error() != "sms: no sender configured" {
		t.Fatalf("err=%v, want no sender configured", err)
	}
}
