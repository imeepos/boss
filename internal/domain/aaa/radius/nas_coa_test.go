package radius

// AAA-A5(G7) CoA per-NAS 密钥回归:正确密钥 ACK、错密钥失败留痕、兼容开关与停用语义。

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"layeh.com/radius"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// startFakeNAS 起一个只认给定密钥的假 NAS(密钥不符静默丢弃,模拟真实设备)。
func startFakeNAS(t *testing.T, secret []byte) int {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		var buff [1500]byte
		for {
			n, addr, err := conn.ReadFrom(buff[:])
			if err != nil {
				return
			}
			if !radius.IsAuthenticRequest(buff[:n], secret) {
				continue
			}
			pkt, err := radius.Parse(buff[:n], secret)
			if err != nil {
				continue
			}
			b, err := pkt.Response(radius.CodeDisconnectACK).Encode()
			if err != nil {
				continue
			}
			_, _ = conn.WriteTo(b, addr)
		}
	}()
	t.Cleanup(func() { conn.Close() })
	return conn.LocalAddr().(*net.UDPAddr).Port
}

func nasStub(secret []byte, port int) nasResolverStub {
	return nasResolverStub{nas: &aaa.NasAuth{Client: aaa.NasClient{Name: "OLT-01", CoAPort: port}, Secret: secret}}
}

func TestNasCoAClientPerNasSecretAndPort(t *testing.T) {
	port := startFakeNAS(t, []byte("nas-key-1"))
	client := NewNasCoAClient(nasStub([]byte("nas-key-1"), port), []byte("global-key"), 39999, 600*time.Millisecond, false)
	if err := client.SendDisconnect(context.Background(), "127.0.0.1", "LOID-C1", "S-1"); err != nil {
		t.Fatalf("per-NAS 密钥应 ACK: %v", err)
	}
}

func TestNasCoAClientWrongSecretFailsWithLog(t *testing.T) {
	port := startFakeNAS(t, []byte("nas-key-1")) // NAS 持有的密钥
	buf := captureStdLog(t)
	client := NewNasCoAClient(nasStub([]byte("wrong-key"), port), nil, 0, 300*time.Millisecond, false)
	if err := client.SendDisconnect(context.Background(), "127.0.0.1", "LOID-C2", "S-2"); err == nil {
		t.Fatal("错密钥应失败(NAS 丢弃无响应)")
	}
	if !strings.Contains(buf.String(), "[aaa] COA SEND FAILED") {
		t.Fatalf("log=%s", buf.String())
	}
}

func TestNasCoAClientCompatAndReject(t *testing.T) {
	globalPort := startFakeNAS(t, []byte("global-key"))
	ctx := context.Background()

	t.Run("兼容开关开启:未注册回退全局密钥与端口", func(t *testing.T) {
		buf := captureStdLog(t)
		client := NewNasCoAClient(nasResolverStub{err: notFoundErr("10.9.9.9")}, []byte("global-key"), globalPort, 600*time.Millisecond, true)
		if err := client.SendDisconnect(ctx, "127.0.0.1", "LOID-C3", "S-3"); err != nil {
			t.Fatalf("兼容回退应 ACK: %v", err)
		}
		if !strings.Contains(buf.String(), "[aaa] COA COMPAT GLOBAL SECRET") {
			t.Fatalf("log=%s", buf.String())
		}
	})
	t.Run("兼容开关关闭:未注册拒绝留痕", func(t *testing.T) {
		buf := captureStdLog(t)
		client := NewNasCoAClient(nasResolverStub{err: notFoundErr("10.9.9.9")}, []byte("global-key"), globalPort, 300*time.Millisecond, false)
		if err := client.SendDisconnect(ctx, "127.0.0.1", "LOID-C4", "S-4"); !errors.Is(err, aaa.ErrNasNotFound) {
			t.Fatalf("err=%v want ErrNasNotFound", err)
		}
		if !strings.Contains(buf.String(), "[aaa] COA NAS REJECT UNREGISTERED") {
			t.Fatalf("log=%s", buf.String())
		}
	})
	t.Run("停用 NAS 即便兼容开启也拒绝", func(t *testing.T) {
		buf := captureStdLog(t)
		client := NewNasCoAClient(nasResolverStub{err: disabledErr()}, []byte("global-key"), globalPort, 300*time.Millisecond, true)
		if err := client.SendDisconnect(ctx, "127.0.0.1", "LOID-C5", "S-5"); !errors.Is(err, aaa.ErrNasDisabled) {
			t.Fatalf("err=%v want ErrNasDisabled", err)
		}
		if !strings.Contains(buf.String(), "[aaa] COA NAS REJECT DISABLED") {
			t.Fatalf("log=%s", buf.String())
		}
	})
}
