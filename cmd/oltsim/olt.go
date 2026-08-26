// OLT Telnet CLI 仿真服务端:镜像 provision.TelnetExecutor 交互面。
// 协议:连接→login:→password:→收 "provision apply template=.. task=.. event=.."→回 OK/ERR。
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
)

// serveTelnet 阻塞服务 Telnet CLI,直至 ctx 取消。
func (s *Sim) serveTelnet(ctx context.Context, ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("oltsim: telnet accept: %v", err)
			continue
		}
		go s.serveConn(ctx, c)
	}
}

// serveConn 处理一次 Telnet 会话(单连接单命令,与 TelnetExecutor 行为对齐)。
func (s *Sim) serveConn(ctx context.Context, c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	if !s.telnetLogin(ctx, c, r) {
		return
	}
	line, err := readLine(ctx, r)
	if err != nil {
		return
	}
	cmd := strings.TrimSpace(line)
	if !strings.HasPrefix(cmd, "provision apply ") {
		s.reply(c, "ERR unknown command")
		s.add(Record{Type: "provision", Result: "ERR", Detail: cmd})
		return
	}
	tpl, task, event := parseApply(cmd)
	if s.ForceErr {
		s.reply(c, "ERR forced failure")
		s.add(Record{Type: "provision", Template: tpl, TaskNo: task, Event: event, Result: "ERR", Detail: "forced"})
		return
	}
	// 完整业务校验:template/task/event 必须齐备(模拟厂家 CLI 参数校验)。
	if tpl == "" || task == "" || event == "" {
		s.reply(c, "ERR invalid apply args")
		s.add(Record{Type: "provision", Template: tpl, TaskNo: task, Event: event, Result: "ERR", Detail: "invalid args"})
		return
	}
	s.reply(c, "OK")
	s.add(Record{Type: "provision", Template: tpl, TaskNo: task, Event: event, Result: "OK", Detail: "applied"})
	log.Printf("oltsim: apply ok template=%s task=%s event=%s", tpl, task, event)
}

// telnetLogin 登录握手:login:→password:,校验凭据;失败注入 deny-login。
func (s *Sim) telnetLogin(ctx context.Context, c net.Conn, r *bufio.Reader) bool {
	_, _ = c.Write([]byte("login:"))
	u, err := readLine(ctx, r)
	if err != nil {
		return false
	}
	_, _ = c.Write([]byte("password:"))
	p, err := readLine(ctx, r)
	if err != nil {
		return false
	}
	if s.DenyLogin || strings.TrimSpace(u) != s.User || strings.TrimSpace(p) != s.Pass {
		_, _ = c.Write([]byte("ERR login denied\n"))
		log.Printf("oltsim: login denied user=%q", strings.TrimSpace(u))
		return false
	}
	return true
}

// reply 回一行应答(带换行,对齐 TelnetExecutor readUntil 的行语义)。
func (s *Sim) reply(c net.Conn, line string) {
	_, _ = c.Write([]byte(line + "\n"))
}

// parseApply 解析 "provision apply template=7 task=PRV-O1 event=preConfigOLT"。
func parseApply(cmd string) (template, task, event string) {
	for _, kv := range strings.Fields(cmd)[2:] { // 跳过 provision apply
		fields := strings.SplitN(kv, "=", 2)
		if len(fields) != 2 {
			continue
		}
		switch fields[0] {
		case "template":
			template = fields[1]
		case "task":
			task = fields[1]
		case "event":
			event = fields[1]
		}
	}
	return template, task, event
}

// readLine 读到换行为止(命令行应答)。
func readLine(ctx context.Context, r *bufio.Reader) (string, error) {
	type res struct {
		line string
		err  error
	}
	ch := make(chan res, 1)
	go func() {
		line, err := r.ReadString('\n')
		ch <- res{strings.TrimSpace(line), err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case rr := <-ch:
		if rr.err != nil && rr.line == "" {
			return "", fmt.Errorf("oltsim: read: %w", rr.err)
		}
		return rr.line, rr.err
	}
}

// readPrompt 逐字节累计直至出现 substr(如 "login:" 无换行 prompt,
// 对齐 provision.readUntil 语义);超时/EOF 报错。
func readPrompt(ctx context.Context, r *bufio.Reader, substr string) (string, error) {
	type res struct {
		buf string
		err error
	}
	ch := make(chan res, 1)
	go func() {
		var buf []byte
		for !strings.Contains(string(buf), substr) {
			b, err := r.ReadByte()
			if err != nil {
				ch <- res{string(buf), err}
				return
			}
			buf = append(buf, b)
		}
		ch <- res{string(buf), nil}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case rr := <-ch:
		if rr.err != nil && !strings.Contains(rr.buf, substr) {
			return "", fmt.Errorf("oltsim: prompt %q: %w", substr, rr.err)
		}
		return rr.buf, nil
	}
}

// listenTCP 监听 TCP 端口(单测注入;生产走 flag)。
func listenTCP(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
