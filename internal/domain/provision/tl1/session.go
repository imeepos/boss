package tl1

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Config 连接与会话参数;零值经 norm 归一化为缺省(PDF §11.3: 10min 空闲断连)。
type Config struct {
	Addr        string // host:13027
	Network     string // "tcp",P2 扩展 "ssl";空取 "tcp"
	User, Pass  string
	Keepalive   time.Duration // 0=3min(空闲断连 10min 的 1/3)
	CmdTimeout  time.Duration // 0=120s,单命令含 DELAY 等待总时长
	DialTimeout time.Duration // 0=10s
}

// norm 归一化零值配置。
func (c Config) norm() Config {
	if c.Network == "" {
		c.Network = "tcp"
	}
	if c.Keepalive <= 0 {
		c.Keepalive = 3 * time.Minute
	}
	if c.CmdTimeout <= 0 {
		c.CmdTimeout = 120 * time.Second
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 10 * time.Second
	}
	return c
}

// dialNet 建连入口;测试注入 fake conn 用。
var dialNet = func(ctx context.Context, network, addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

// Dial 建连并 LOGIN;鉴权被拒返回 ErrAuth,连接层失败返回 ErrConnBroken。
func Dial(ctx context.Context, cfg Config) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg = cfg.norm()
	conn, err := dialNet(ctx, cfg.Network, cfg.Addr, cfg.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("%w: dial %s: %v", ErrConnBroken, cfg.Addr, err)
	}
	s := &Session{conn: conn, cfg: cfg, stopCh: make(chan struct{})}
	if err := s.login(ctx); err != nil {
		conn.Close()
		return nil, err
	}
	s.startKeepalive()
	return s, nil
}

// Session 单连接会话状态机:命令串行、ctag 自增、空闲握手保活(PDF §11)。
type Session struct {
	conn net.Conn
	cfg  Config

	mu      sync.Mutex // 串行化 Do 与 SHAKEHAND 的写读
	ctag    int
	closed  bool
	stopCh  chan struct{}
	stopOne sync.Once
	wg      sync.WaitGroup // keepalive goroutine
}

// login 发 LOGIN 并要求 COMPLD 且 EN=0,否则 ErrAuth(PDF §11.1)。
func (s *Session) login(ctx context.Context) error {
	line, err := Build(Command{Verb: "LOGIN", Payload: []KV{{"UN", s.cfg.User}, {"PWD", s.cfg.Pass}}}, "1")
	if err != nil {
		return err
	}
	if err := s.write(line); err != nil {
		return err
	}
	resp, err := s.readResponse(ctx, time.Now().Add(s.cfg.CmdTimeout))
	if err != nil {
		return err
	}
	if resp.Completion == "COMPLD" && resp.EN == 0 {
		return nil
	}
	return fmt.Errorf("%w: ctag=%s completion=%s EN=%d ENDESC=%s", ErrAuth, resp.CTag, resp.Completion, resp.EN, resp.ENDESC)
}

// write 带写超时写出一条已构建报文。
func (s *Session) write(line []byte) error {
	s.conn.SetWriteDeadline(time.Now().Add(s.cfg.CmdTimeout))
	if _, err := s.conn.Write(line); err != nil {
		return fmt.Errorf("%w: write: %v", ErrConnBroken, err)
	}
	return nil
}

// readResponse 读一帧并解析;读超时/EOF/残断一律归 ErrConnBroken 供上层重连判定。
func (s *Session) readResponse(ctx context.Context, deadline time.Time) (*Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.conn.SetReadDeadline(deadline)
	frame, err := ReadFrame(s.conn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnBroken, err)
	}
	return Parse(frame)
}

// Do 串行执行一条命令:自增 ctag 写出,读到同 ctag 终止帧返回。
// DELAY 继续追帧;DENY/PRTL/RTRV 原样透传由调用方判定;总时长受 CmdTimeout 与 ctx 截止约束。
func (s *Session) Do(ctx context.Context, c Command) (*Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("%w: session closed", ErrConnBroken)
	}
	ctag := s.nextCtagLocked()
	line, err := Build(c, ctag)
	if err != nil {
		return nil, err
	}
	if err := s.write(line); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(s.cfg.CmdTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	for {
		resp, err := s.readResponse(ctx, deadline)
		if err != nil {
			return nil, err
		}
		if resp.CTag != ctag {
			continue // 异 ctag 残帧丢弃
		}
		switch resp.Completion {
		case "DELAY":
			continue
		case "COMPLD", "DENY", "PRTL", "RTRV":
			return resp, nil
		default:
			return nil, fmt.Errorf("%w: completion=%q", ErrParse, resp.Completion)
		}
	}
}

// nextCtagLocked 自增 ctag,调用方须持 mu。
func (s *Session) nextCtagLocked() string {
	s.ctag++
	return fmt.Sprintf("B%06d", s.ctag)
}

// startKeepalive 空闲握手后台协程:Do 持 mu 期间自然不发,停于 stopCh。
func (s *Session) startKeepalive() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		t := time.NewTicker(s.cfg.Keepalive)
		defer t.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-t.C:
				s.shake()
			}
		}
	}()
}

// shake 持锁发一条 SHAKEHAND;写失败仅留痕,断连由下次 Do 自愈。
func (s *Session) shake() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	line, err := Build(Command{Verb: "SHAKEHAND"}, s.nextCtagLocked())
	if err != nil {
		return
	}
	if err := s.write(line); err != nil {
		log.Printf("[provision-tl1] SHAKEHAND FAILED addr=%s: %v", s.cfg.Addr, err)
	}
}

// Close 发 LOGOUT(尽力而为)后停握手、关连接;幂等。
func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.stopOne.Do(func() { close(s.stopCh) })
	if line, err := Build(Command{Verb: "LOGOUT"}, s.nextCtagLocked()); err == nil {
		s.write(line) // 尽力而为,断连不阻断关闭
	}
	s.mu.Unlock()
	s.wg.Wait()
	return s.conn.Close()
}
