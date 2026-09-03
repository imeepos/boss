package provision

// TelnetExecutor 单测:fake OLT(TCP 行协议)验证登录/下发/OK 校验与失败留痕语义。

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeOLT 极简 OLT 模拟器:login:→password:→读 apply 命令→回 OK/ERR。
type fakeOLT struct {
	ln    net.Listener
	reply string // OK 或其它
	cmds  []string
	done  chan struct{}
}

func startFakeOLT(t *testing.T, reply string) *fakeOLT {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeOLT{ln: ln, reply: reply, done: make(chan struct{})}
	go f.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return f
}

func (f *fakeOLT) serve() {
	defer close(f.done)
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		r := bufio.NewReader(c)
		_, _ = c.Write([]byte("login:"))
		_, _ = r.ReadString('\n')
		_, _ = c.Write([]byte("password:"))
		_, _ = r.ReadString('\n')
		cmd, _ := r.ReadString('\n')
		f.cmds = append(f.cmds, strings.TrimSpace(cmd))
		_, _ = c.Write([]byte(f.reply + "\n"))
		_ = c.Close()
	}
}

func (f *fakeOLT) addr() string { return f.ln.Addr().String() }

func TestTelnetExecutor_Exec(t *testing.T) {
	ctx := context.Background()
	task := Task{ID: 1, TaskNo: "TASK-1", StageEvent: "preConfigOLT", TemplateID: 7}

	t.Run("下发成功", func(t *testing.T) {
		f := startFakeOLT(t, "OK")
		e := &TelnetExecutor{Addr: f.addr(), User: "admin", Pass: "secret", Timeout: 2 * time.Second}
		trace, err := e.Exec(ctx, task)
		if err != nil {
			t.Fatalf("Exec: %v", err)
		}
		if len(trace.Commands) != 1 || !strings.Contains(trace.Commands[0], "template=7") || !strings.Contains(trace.Commands[0], "task=TASK-1") {
			t.Fatalf("commands=%v", trace.Commands)
		}
		if trace.Response != "OK" {
			t.Fatalf("response=%q", trace.Response)
		}
		if trace.Driver != DriverTelnet {
			t.Fatalf("driver=%q, want telnet", trace.Driver)
		}
	})

	t.Run("OLT 应答非 OK", func(t *testing.T) {
		f := startFakeOLT(t, "ERR config invalid")
		e := &TelnetExecutor{Addr: f.addr(), User: "admin", Pass: "secret", Timeout: 2 * time.Second}
		trace, err := e.Exec(ctx, task)
		if err == nil || !strings.Contains(err.Error(), "nok") {
			t.Fatalf("err=%v, want nok", err)
		}
		if trace.Response != "ERR config invalid" {
			t.Fatalf("response=%q", trace.Response)
		}
	})

	t.Run("应答含 OK 但非完整 OK 行", func(t *testing.T) {
		f := startFakeOLT(t, "OK applied")
		e := &TelnetExecutor{Addr: f.addr(), User: "admin", Pass: "secret", Timeout: 2 * time.Second}
		trace, err := e.Exec(ctx, task)
		if err == nil || !strings.Contains(err.Error(), "nok") {
			t.Fatalf("err=%v, want nok", err)
		}
		if trace.Response != "OK applied" {
			t.Fatalf("response=%q", trace.Response)
		}
	})

	t.Run("连接失败", func(t *testing.T) {
		e := &TelnetExecutor{Addr: "127.0.0.1:1", User: "admin", Pass: "secret", Timeout: time.Second}
		if _, err := e.Exec(ctx, task); err == nil {
			t.Fatal("want dial error")
		}
	})
}
