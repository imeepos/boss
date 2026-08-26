// oltsim Telnet CLI 服务端单测:登录握手/命令解析/回执语义/故障注入。
package main

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// dialApply 起一个临时 Sim 并执行一次完整握手+下发,返回回执与台账。
func dialApply(t *testing.T, s *Sim, user, pass, cmd string) (string, error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go s.serveTelnet(context.Background(), ln)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	r := bufio.NewReader(c)
	expectPrompt(t, ctx, r, "login:")
	_, _ = c.Write([]byte(user + "\n"))
	expectPrompt(t, ctx, r, "password:")
	_, _ = c.Write([]byte(pass + "\n"))
	_, _ = c.Write([]byte(cmd + "\n"))
	line, err := readLine(ctx, r)
	if err != nil {
		return "", err
	}
	// 台账追加在 reply 之后的 goroutine 内;轮询等稳定(≤1s)。
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(s.all()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return line, nil
}

func expectPrompt(t *testing.T, ctx context.Context, r *bufio.Reader, want string) {
	t.Helper()
	line, err := readPrompt(ctx, r, want)
	if err != nil {
		t.Fatalf("read prompt %q: %v", want, err)
	}
	if !strings.Contains(line, want) {
		t.Fatalf("prompt=%q, want contains %q", line, want)
	}
}

func TestSim_TelnetApplyOK(t *testing.T) {
	s := &Sim{User: "admin", Pass: "secret"}
	reply, err := dialApply(t, s, "admin", "secret",
		"provision apply template=7 task=PRV-O1 event=preConfigOLT")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "OK" {
		t.Fatalf("reply=%q, want OK", reply)
	}
	recs := s.all()
	if len(recs) != 1 || recs[0].Result != "OK" || recs[0].TaskNo != "PRV-O1" ||
		recs[0].Event != "preConfigOLT" || recs[0].Template != "7" {
		t.Fatalf("recs=%+v", recs)
	}
}

func TestSim_TelnetDenyLogin(t *testing.T) {
	s := &Sim{User: "admin", Pass: "secret", DenyLogin: true}
	reply, err := dialApply(t, s, "admin", "secret",
		"provision apply template=7 task=T event=preConfigOLT")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "login denied") {
		t.Fatalf("reply=%q, want login denied", reply)
	}
	if len(s.all()) != 0 {
		t.Fatalf("denied login must not record, got %+v", s.all())
	}
}

func TestSim_TelnetWrongPassword(t *testing.T) {
	s := &Sim{User: "admin", Pass: "secret"}
	reply, err := dialApply(t, s, "admin", "wrong",
		"provision apply template=7 task=T event=preConfigOLT")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "login denied") {
		t.Fatalf("reply=%q, want login denied", reply)
	}
}

func TestSim_TelnetForceErr(t *testing.T) {
	s := &Sim{User: "admin", Pass: "secret", ForceErr: true}
	reply, err := dialApply(t, s, "admin", "secret",
		"provision apply template=7 task=PRV-O1 event=preConfigOLT")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(reply, "ERR") {
		t.Fatalf("reply=%q, want ERR", reply)
	}
	recs := s.all()
	if len(recs) != 1 || recs[0].Result != "ERR" {
		t.Fatalf("recs=%+v", recs)
	}
}

func TestSim_TelnetUnknownCommand(t *testing.T) {
	s := &Sim{User: "admin", Pass: "secret"}
	reply, err := dialApply(t, s, "admin", "secret", "show version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply, "unknown command") {
		t.Fatalf("reply=%q, want unknown command", reply)
	}
}

func TestParseApply(t *testing.T) {
	tpl, task, event := parseApply("provision apply template=9 task=PRV-88 event=activateUser")
	if tpl != "9" || task != "PRV-88" || event != "activateUser" {
		t.Fatalf("tpl=%q task=%q event=%q", tpl, task, event)
	}
}
