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
