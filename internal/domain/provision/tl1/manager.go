package tl1

import (
	"context"
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// retryBackoff 断线重连前的退避延迟;var 便于测试缩短。
var retryBackoff = time.Second

// breakerWindow 鉴权失败熔断窗口,窗口内不重试直返 ErrAuth。
const breakerWindow = 5 * time.Minute

// Manager 端点级会话持有者:懒建连,断线退避重连一次,鉴权失败熔断。
type Manager struct {
	cfg        Config
	mu         sync.Mutex
	sess       *Session
	authFailAt time.Time
}

// NewManager 归一化配置并构造端点管理器。
func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg.norm()}
}

// Use 切换端点并清理旧会话/鉴权熔断；同端点调用仍会重建会话。
func (m *Manager) Use(endpoint Endpoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sess != nil {
		_ = m.sess.Close()
		m.sess = nil
	}
	m.authFailAt = time.Time{}
	m.cfg.Addr = net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port))
	m.cfg.User, m.cfg.Pass = endpoint.User, endpoint.Pass
}

// Endpoint 返回当前端点快照,供测试/诊断使用。
func (m *Manager) Endpoint() Endpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	host, port, err := net.SplitHostPort(m.cfg.Addr)
	if err != nil {
		return Endpoint{Host: m.cfg.Addr, User: m.cfg.User, Pass: m.cfg.Pass}
	}
	p, _ := strconv.Atoi(port)
	return Endpoint{Host: host, Port: p, User: m.cfg.User, Pass: m.cfg.Pass}
}

// Do 懒建连后执行一条命令;断线退避后重连一次再试,不递归;
// DENY 等业务结果原样透传,鉴权失败经建连路径熔断。
func (m *Manager) Do(ctx context.Context, c Command) (*Response, error) {
	sess, err := m.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := sess.Do(ctx, c)
	if !errors.Is(err, ErrConnBroken) {
		return resp, err
	}
	m.drop(sess)
	if err := backoff(ctx); err != nil {
		return nil, err
	}
	sess2, err := m.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	return sess2.Do(ctx, c)
}

// WithSession 多命令原子段:整段持锁执行,断线退避重连一次后整段重跑,
// 配合 executor 先查后写的幂等性保证不留半程配置。
func (m *Manager) WithSession(ctx context.Context, fn func(*Session) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.inBreaker() {
		return ErrAuth
	}
	if m.sess == nil {
		s, err := m.dialLocked(ctx)
		if err != nil {
			return err
		}
		m.sess = s
	}
	if err := fn(m.sess); !errors.Is(err, ErrConnBroken) {
		return err
	}
	m.sess.Close()
	m.sess = nil
	if err := backoff(ctx); err != nil {
		return err
	}
	s, err := m.dialLocked(ctx)
	if err != nil {
		return err
	}
	m.sess = s
	return fn(s)
}

// ensureSession 取当前会话,没有则建;持锁串行化建连。
func (m *Manager) ensureSession(ctx context.Context) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sess != nil {
		return m.sess, nil
	}
	if m.inBreaker() {
		return nil, ErrAuth
	}
	s, err := m.dialLocked(ctx)
	if err != nil {
		return nil, err
	}
	m.sess = s
	return s, nil
}

// dialLocked 建连;鉴权失败记熔断点并留可 grep 告警日志。
func (m *Manager) dialLocked(ctx context.Context) (*Session, error) {
	s, err := Dial(ctx, m.cfg)
	if err != nil {
		if errors.Is(err, ErrAuth) {
			m.authFailAt = time.Now()
			log.Printf("[provision-tl1] LOGIN FAILED addr=%s: %v", m.cfg.Addr, err)
		}
		return nil, err
	}
	return s, nil
}

// drop 摘除失效会话并关闭,连接层失败自愈入口。
func (m *Manager) drop(sess *Session) {
	m.mu.Lock()
	if m.sess == sess {
		m.sess = nil
	}
	m.mu.Unlock()
	sess.Close()
}

// inBreaker 熔断窗口内返回 true。
func (m *Manager) inBreaker() bool {
	return !m.authFailAt.IsZero() && time.Since(m.authFailAt) < breakerWindow
}

// backoff 退避等待,ctx 取消即时返回。
func backoff(ctx context.Context) error {
	t := time.NewTimer(retryBackoff)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
