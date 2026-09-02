// Package sim 是 U2000 北向 TL1 接口的进程内仿真器,供 executor/集成测试与
// cmd/tl1sim 命令行包装使用;测试基建,不进生产部署。
package sim

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ymm-001/boss/internal/domain/provision/tl1"
)

// Options 仿真器启动参数。
type Options struct {
	Addr         string // 监听地址,空取 127.0.0.1:0
	User, Pass   string // LOGIN 凭据
	ONUOperState string // 新增 ONU 的 OperState: UP|Power-Off|LOS(环节10 负路径注入)
	DelayMs      int    // >0 时每条指令先回 DELAY 帧并延迟后回最终响应
	DenyNext     string // 对下一条该动词的指令 DENY 一次(单次故障注入)
	RecordPath   string // 指令留痕 JSONL;空不落盘
}

// Server 可 Start/Stop 的 TL1 仿真网元;测试进程内直接构造。
type Server struct {
	opt    Options
	st     *state
	rec    *record
	ln     net.Listener
	mu     sync.Mutex
	conns  map[net.Conn]struct{}
	closed bool
	wg     sync.WaitGroup
}

// New 创建并监听;Start 前不接受连接。
func New(opt Options) (*Server, error) {
	if opt.Addr == "" {
		opt.Addr = "127.0.0.1:0"
	}
	if opt.ONUOperState == "" {
		opt.ONUOperState = "UP"
	}
	rec, err := newRecord(opt.RecordPath)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", opt.Addr)
	if err != nil {
		_ = rec.close()
		return nil, err
	}
	s := &Server{opt: opt, rec: rec, ln: ln, conns: map[net.Conn]struct{}{}}
	s.st = newState(opt, rec)
	return s, nil
}

// Addr 实际监听地址(测试用 :0 随机端口)。
func (s *Server) Addr() string { return s.ln.Addr().String() }

// Start 启动 accept 循环。
func (s *Server) Start() {
	s.wg.Add(1)
	go s.acceptLoop()
	log.Printf("[tl1sim] listening %s user=%s oper-state=%s", s.Addr(), s.opt.User, s.opt.ONUOperState)
}

// Stop 停止监听、断开全部连接并关留痕文件;幂等。
func (s *Server) Stop() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()
	_ = s.ln.Close()
	s.DropConns()
	s.wg.Wait()
	return s.rec.close()
}

// DropConns 模拟网元侧断线:关闭全部既有连接,监听保持,新连接照常受理。
func (s *Server) DropConns() {
	for _, c := range s.connsSnapshot() {
		_ = c.Close()
	}
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.track(conn, true)
		s.wg.Add(1)
		go s.serve(conn)
	}
}

// serve 单连接循环:逐帧读取→状态机处理→按序写出响应帧。
func (s *Server) serve(conn net.Conn) {
	defer s.wg.Done()
	defer s.track(conn, false)
	for {
		frame, err := tl1.ReadFrame(conn)
		if err != nil {
			return
		}
		frames, closeConn := s.st.handle(frame)
		for _, f := range frames {
			if _, err := conn.Write([]byte(f)); err != nil {
				return
			}
		}
		if closeConn {
			return
		}
	}
}

func (s *Server) track(conn net.Conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		s.conns[conn] = struct{}{}
	} else {
		delete(s.conns, conn)
	}
}

func (s *Server) connsSnapshot() []net.Conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]net.Conn, 0, len(s.conns))
	for c := range s.conns {
		out = append(out, c)
	}
	return out
}

// record JSONL 指令留痕;path 为空时为空操作。
type record struct {
	mu sync.Mutex
	f  *os.File
}

func newRecord(path string) (*record, error) {
	if path == "" {
		return &record{}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &record{f: f}, nil
}

// add 追加一条指令留痕;失败仅告警不阻断仿真(留痕是辅助通道)。
func (r *record) add(verb, raw string) {
	if r == nil || r.f == nil {
		return
	}
	line, err := json.Marshal(map[string]string{
		"ts":   time.Now().UTC().Format(time.RFC3339),
		"verb": verb,
		"cmd":  raw,
	})
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.f.Write(append(line, '\n')); err != nil {
		log.Printf("[tl1sim] RECORD FAILED verb=%s: %v", verb, err)
	}
}

func (r *record) close() error {
	if r == nil || r.f == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	err := r.f.Close()
	r.f = nil
	return err
}
