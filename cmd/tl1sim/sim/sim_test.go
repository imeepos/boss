package sim

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// newTestServer 进程内起仿真器,返回实例与生效配置。
func newTestServer(t *testing.T, mod func(*Options)) (*Server, Options) {
	t.Helper()
	opt := Options{
		Addr:       "127.0.0.1:0",
		User:       "admin",
		Pass:       "admin",
		RecordPath: filepath.Join(t.TempDir(), "sim.jsonl"),
	}
	if mod != nil {
		mod(&opt)
	}
	srv, err := New(opt)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.Start()
	t.Cleanup(func() { _ = srv.Stop() })
	return srv, opt
}

// TestSimLoginBadPass 登录错密码→DENY EN=76546031,Manager 报 ErrAuth。
func TestSimLoginBadPass(t *testing.T) {
	srv, opt := newTestServer(t, nil)
	_, err := tl1.Dial(context.Background(), tl1.Config{
		Addr: srv.Addr(), User: opt.User, Pass: "bad",
		CmdTimeout: 5 * time.Second, DialTimeout: 2 * time.Second,
	})
	if !errors.Is(err, tl1.ErrAuth) {
		t.Fatalf("want ErrAuth got %v", err)
	}
}

// TestSimProtocolRegression ADD-ONU 重复 DENY、LST 表格可被 codec 解析、未登录 DENY。
func TestSimProtocolRegression(t *testing.T) {
	srv, opt := newTestServer(t, nil)
	s, err := tl1.Dial(context.Background(), tl1.Config{
		Addr: srv.Addr(), User: opt.User, Pass: opt.Pass,
		CmdTimeout: 5 * time.Second, DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	ctx := context.Background()
	acc := []tl1.KV{{K: "OLTID", V: "10.0.0.9"}, {K: "PONID", V: "NA-0-7-5"}}
	pay := []tl1.KV{
		{K: "AUTHTYPE", V: "LOID"}, {K: "ONUID", V: "loid-1"}, {K: "ONUNO", V: "3"},
		{K: "DESC", V: "PRV-1"}, {K: "ONUTYPE", V: "FTTH"},
	}
	resp, err := s.Do(ctx, tl1.Command{Verb: "ADD-ONU", Tag: "ADDONT", Access: acc, Payload: pay})
	if err != nil || resp.Completion != "COMPLD" {
		t.Fatalf("ADD-ONU: %v %+v", err, resp)
	}
	resp, err = s.Do(ctx, tl1.Command{Verb: "ADD-ONU", Tag: "ADDONT", Access: acc, Payload: pay})
	if err != nil || resp.Completion != "DENY" || resp.EN != enONUExists {
		t.Fatalf("dup ADD-ONU: %v %+v", err, resp)
	}
	qry := append(acc, tl1.KV{K: "ONUIDTYPE", V: "LOID"}, tl1.KV{K: "ONUID", V: "loid-1"})
	resp, err = s.Do(ctx, tl1.Command{Verb: "LST-ONU", Access: qry})
	if err != nil || len(resp.Rows) != 1 {
		t.Fatalf("LST-ONU: %v rows=%d", err, len(resp.Rows))
	}
	if resp.Rows[0]["LOID"] != "loid-1" || resp.Rows[0]["ONUNO"] != "3" {
		t.Fatalf("row=%+v", resp.Rows[0])
	}
	_ = s.Close()
}

// TestSimRequiresLogin 未登录命令一律 DENY(独立裸连接验证)。
func TestSimRequiresLogin(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("LST-ONU::OLTID=x,PONID=y:B1::;")); err != nil {
		t.Fatalf("write: %v", err)
	}
	frame, err := tl1.ReadFrame(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	resp, err := tl1.Parse(frame)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Completion != "DENY" || resp.EN != enNotLoggedIn {
		t.Fatalf("resp=%+v want DENY EN=%d", resp, enNotLoggedIn)
	}
}
