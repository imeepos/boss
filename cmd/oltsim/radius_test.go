// 光猫上线 RADIUS 客户端单测:fake boss-aaa 服务端回执验证帧构造与带宽解析。
package main

import (
	"context"
	"net"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

// fakeRadiusAAA 极简 RADIUS 服务端:按 User-Name(LOID)回 Access-Accept/Reject。
type fakeRadiusAAA struct {
	ln     net.PacketConn
	secret string
	bw     string // 命中的 Access-Accept 带宽(FramedPool)
	reject bool
}

func startFakeRadiusAAA(t *testing.T, secret, bw string, reject bool) *fakeRadiusAAA {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeRadiusAAA{ln: pc, secret: secret, bw: bw, reject: reject}
	go f.serve(t)
	t.Cleanup(func() { _ = pc.Close() })
	return f
}

func (f *fakeRadiusAAA) addr() string { return f.ln.LocalAddr().String() }

func (f *fakeRadiusAAA) serve(t *testing.T) {
	buf := make([]byte, 4096)
	for {
		n, addr, err := f.ln.ReadFrom(buf)
		if err != nil {
			return
		}
		pkt, err := radius.Parse(buf[:n], []byte(f.secret))
		if err != nil {
			t.Logf("fake radius parse: %v", err)
			continue
		}
		resp := pkt.Response(radius.CodeAccessAccept)
		if f.reject {
			resp = pkt.Response(radius.CodeAccessReject)
		} else {
			_ = rfc2869.FramedPool_SetString(resp, f.bw)
			_ = rfc2865.SessionTimeout_Set(resp, 3600)
		}
		enc, err := resp.Encode()
		if err != nil {
			t.Logf("fake radius encode: %v", err)
			continue
		}
		if _, err := f.ln.WriteTo(enc, addr); err != nil {
			return
		}
	}
}

func TestSim_RadiusAuthAccept(t *testing.T) {
	srv := startFakeRadiusAAA(t, "boss-aaa-secret", "300M/150M", false)
	s := &Sim{RadiusAddr: srv.addr(), RadiusSecret: "boss-aaa-secret"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	code, bw, err := s.radiusAuth(ctx, "LOID-88A1")
	if err != nil {
		t.Fatalf("radiusAuth: %v", err)
	}
	if code != radius.CodeAccessAccept {
		t.Fatalf("code=%v, want AccessAccept", code)
	}
	if bw != "300M/150M" {
		t.Fatalf("bandwidth=%q, want 300M/150M(套餐带宽生效)", bw)
	}
}

func TestSim_RadiusAuthReject(t *testing.T) {
	srv := startFakeRadiusAAA(t, "boss-aaa-secret", "", true)
	s := &Sim{RadiusAddr: srv.addr(), RadiusSecret: "boss-aaa-secret"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	code, _, err := s.radiusAuth(ctx, "LOID-NOPE")
	if err != nil {
		t.Fatalf("radiusAuth: %v", err)
	}
	if code != radius.CodeAccessReject {
		t.Fatalf("code=%v, want AccessReject", code)
	}
}
