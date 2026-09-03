package provision

// TelnetExecutor 真实 OLT 配置下发执行器(债务偿还:替换 cmd/provisioner 的 logExecutor 桩)。
// 协议:TCP 行协议(login:/password: 提示 → 下发 apply 命令 → 期望 OK 应答);
// 与厂商 CLI 的适配在命令模板层完成,此处固化连接/鉴权/一次下发的协议骨架。

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// TelnetExecutor 通过 TCP 行协议向 OLT 下发模板配置。
type TelnetExecutor struct {
	Addr    string // host:port
	User    string
	Pass    string
	Timeout time.Duration
	// Dial 连接构造(单测注入 fake OLT;缺省走 net.DialTimeout)。
	Dial func(ctx context.Context, addr string) (net.Conn, error)
}

// defaultTelnetTimeout 缺省单次交互超时。
const defaultTelnetTimeout = 5 * time.Second

// Exec 执行一次下发:连接 → 登录 → 下发命令 → 校验 OK;回传指令与设备应答留痕。
func (e *TelnetExecutor) Exec(ctx context.Context, t Task) (ExecTrace, error) {
	trace := ExecTrace{Commands: []string{}, Driver: DriverTelnet}
	dial := e.Dial
	if dial == nil {
		dial = func(c context.Context, a string) (net.Conn, error) {
			return net.DialTimeout("tcp", a, e.timeout())
		}
	}
	conn, err := dial(ctx, e.Addr)
	if err != nil {
		return trace, fmt.Errorf("provision: telnet dial: %w", err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	if err := e.chat(ctx, conn, r, "login:", e.User); err != nil {
		return trace, err
	}
	if err := e.chat(ctx, conn, r, "password:", e.Pass); err != nil {
		return trace, err
	}
	cmd := fmt.Sprintf("provision apply template=%d task=%s event=%s", t.TemplateID, t.TaskNo, t.StageEvent)
	trace.Commands = append(trace.Commands, cmd)
	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return trace, fmt.Errorf("provision: telnet cmd: %w", err)
	}
	line, err := readUntil(ctx, r, "OK", e.timeout())
	trace.Response = strings.TrimSpace(line)
	if err != nil {
		return trace, fmt.Errorf("provision: telnet read: %w", err)
	}
	if !strings.Contains(line, "OK") {
		return trace, fmt.Errorf("provision: telnet nok: %s", trace.Response)
	}
	return trace, nil
}

// chat 等待提示串后回送一行;提示串可能无换行,按子串匹配而非整行读取。
func (e *TelnetExecutor) chat(ctx context.Context, conn net.Conn, r *bufio.Reader, prompt, value string) error {
	line, err := readUntil(ctx, r, prompt, e.timeout())
	if err != nil {
		return fmt.Errorf("provision: telnet prompt: %w", err)
	}
	if !strings.Contains(line, prompt) {
		return fmt.Errorf("provision: telnet unexpected prompt: %s", strings.TrimSpace(line))
	}
	if _, err := conn.Write([]byte(value + "\n")); err != nil {
		return fmt.Errorf("provision: telnet send: %w", err)
	}
	return nil
}

// timeout 归一化超时;0 或负用缺省。
func (e *TelnetExecutor) timeout() time.Duration {
	if e.Timeout <= 0 {
		return defaultTelnetTimeout
	}
	return e.Timeout
}

// readUntil 累计读取直至出现 substr 或超时;返回已读内容。
func readUntil(ctx context.Context, r *bufio.Reader, substr string, d time.Duration) (string, error) {
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
				if len(buf) > 0 {
					ch <- res{string(buf), nil}
					return
				}
				ch <- res{"", err}
				return
			}
			buf = append(buf, b)
			if b == '\n' { // 到行尾即返回,不继续等 substr(供 nok 判定)
				break
			}
		}
		ch <- res{string(buf), nil}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(d):
		return "", fmt.Errorf("provision: telnet timeout")
	case rr := <-ch:
		return rr.buf, rr.err
	}
}
