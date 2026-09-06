package radius

// AAA-A5(G7) per-NAS 密钥校验回归:SecretSource 分流留痕 + 真实 UDP 收发。

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// nasResolverStub 固定结果的注册表桩。
type nasResolverStub struct {
	nas *aaa.NasAuth
	err error
}

func (s nasResolverStub) LookupNas(context.Context, string) (*aaa.NasAuth, error) {
	return s.nas, s.err
}

func notFoundErr(ip string) error { return fmt.Errorf("%w: ip=%s", aaa.ErrNasNotFound, ip) }

func disabledErr() error { return fmt.Errorf("%w: name=OLT-01 ip=10.0.0.9", aaa.ErrNasDisabled) }

func TestRegistrySecretSourcePaths(t *testing.T) {
	hit := &aaa.NasAuth{Client: aaa.NasClient{Name: "OLT-01", Vendor: aaa.NasVendorHuawei}, Secret: []byte("per-nas-secret")}
	cases := []struct {
		name       string
		stub       nasResolverStub
		compat     bool
		wantSecret string
		wantErr    error
		logNeed    string
	}{
		{name: "注册命中返回 per-NAS 密钥", stub: nasResolverStub{nas: hit}, wantSecret: "per-nas-secret"},
		{name: "未注册拒绝留痕", stub: nasResolverStub{err: notFoundErr("10.9.9.9")}, wantErr: aaa.ErrNasNotFound, logNeed: "[aaa] NAS REJECT UNREGISTERED"},
		{name: "停用拒绝留痕(兼容开关不救停用)", stub: nasResolverStub{err: disabledErr()}, compat: true, wantErr: aaa.ErrNasDisabled, logNeed: "[aaa] NAS REJECT DISABLED"},
		{name: "查询故障 fail-closed 留痕", stub: nasResolverStub{err: errors.New("pg down")}, wantErr: nil, logNeed: "[aaa] NAS LOOKUP FAILED"},
		{name: "兼容开关开启未注册回退全局密钥", stub: nasResolverStub{err: notFoundErr("10.9.9.9")}, compat: true, wantSecret: "global-secret"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf := captureStdLog(t)
			src := &RegistrySecretSource{Registry: c.stub, Global: []byte("global-secret"), Compat: c.compat}
			secret, err := src.RADIUSSecret(context.Background(), remoteAt("10.9.9.9"))
			if c.wantErr != nil {
				if !errors.Is(err, c.wantErr) {
					t.Fatalf("err=%v want %v", err, c.wantErr)
				}
			} else if c.wantSecret == "" && err == nil {
				t.Fatal("want error")
			}
			if c.wantSecret != "" {
				if err != nil {
					t.Fatalf("err=%v", err)
				}
				if string(secret) != c.wantSecret {
					t.Fatalf("secret=%q want %q", secret, c.wantSecret)
				}
			}
			if c.logNeed != "" && !strings.Contains(buf.String(), c.logNeed) {
				t.Fatalf("log=%s want %s", buf.String(), c.logNeed)
			}
		})
	}
}

func remoteAt(ip string) net.Addr { return &net.TCPAddr{IP: net.ParseIP(ip), Port: 1812} }

// captureStdLog 捕获标准 log 输出(留痕断言用;radius 包本地副本)。
func captureStdLog(t *testing.T) *strings.Builder {
	t.Helper()
	var buf strings.Builder
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

// startTestRadius 以给定 SecretSource 起真实 UDP 服务(127.0.0.1 随机端口)。
func startTestRadius(t *testing.T, src radius.SecretSource) (string, *strings.Builder) {
	t.Helper()
	var buf strings.Builder
	srv := &radius.PacketServer{
		Handler: &Handler{Auth: &authStub{decision: aaa.Decision{Authorize: true, Bandwidth: "100M", SessionTTL: 60}}},
		SecretSource: src,
		ErrorLog:     log.New(&buf, "", 0),
	}
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(conn) }()
	t.Cleanup(func() { conn.Close() })
	return conn.LocalAddr().String(), &buf
}

func exchangeAccess(t *testing.T, addr, secret, loid string) (*radius.Packet, error) {
	t.Helper()
	pkt := radius.New(radius.CodeAccessRequest, []byte(secret))
	_ = rfc2865.UserName_SetString(pkt, loid)
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	return (&radius.Client{}).Exchange(ctx, pkt, addr)
}

func TestPerNasSecretEndToEnd(t *testing.T) {
	hit := &aaa.NasAuth{Client: aaa.NasClient{Name: "OLT-01"}, Secret: []byte("per-nas-secret")}
	addr, errLog := startTestRadius(t, &RegistrySecretSource{Registry: nasResolverStub{nas: hit}})

	t.Run("per-NAS 密钥正确放行", func(t *testing.T) {
		resp, err := exchangeAccess(t, addr, "per-nas-secret", "LOID-OK")
		if err != nil {
			t.Fatalf("exchange: %v", err)
		}
		if resp.Code != radius.CodeAccessAccept {
			t.Fatalf("code=%v", resp.Code)
		}
	})
	t.Run("计费错密钥拒绝(库层丢弃留痕)", func(t *testing.T) {
		// RFC 2865:Access-Request 的 Request Authenticator 为随机数,服务端不可校验;
		// Accounting-Request 的 Authenticator=MD5(报文+密钥),错密钥在库层可判定并丢弃。
		pkt := radius.New(radius.CodeAccountingRequest, []byte("wrong-secret"))
		_ = rfc2866.AcctStatusType_Set(pkt, rfc2866.AcctStatusType_Value_Stop)
		ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
		defer cancel()
		if _, err := (&radius.Client{}).Exchange(ctx, pkt, addr); err == nil {
			t.Fatal("错密钥应无响应")
		}
		if !strings.Contains(errLog.String(), "bad secret") {
			t.Fatalf("errlog=%s", errLog.String())
		}
	})
}

func TestUnregisteredNasDroppedWithLog(t *testing.T) {
	addr, errLog := startTestRadius(t, &RegistrySecretSource{Registry: nasResolverStub{err: notFoundErr("127.0.0.1")}})
	stdLog := captureStdLog(t)
	if _, err := exchangeAccess(t, addr, "any-secret", "LOID-X"); err == nil {
		t.Fatal("未注册 NAS 应无响应")
	}
	if !strings.Contains(stdLog.String(), "[aaa] NAS REJECT UNREGISTERED") {
		t.Fatalf("stdlog=%s", stdLog.String())
	}
	if !strings.Contains(errLog.String(), "secret source") {
		t.Fatalf("errlog=%s", errLog.String())
	}
}