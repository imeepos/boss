package radius

import (
	"context"
	"log"
	"os"

	"layeh.com/radius"
)

// Server 封装 radius.PacketServer 生命周期与优雅退出。
type Server struct {
	srv  *radius.PacketServer
	done chan struct{}
}

// New 创建 RADIUS 服务;addr 形如 ":1812"(认证)或 ":1813"(计费);
// secret 为全局共享密钥(兼容单 NAS 部署与测试;A5 生产路径走 NewWithSource)。
func New(addr string, secret []byte, h radius.Handler) *Server {
	return NewWithSource(addr, radius.StaticSecretSource(secret), h)
}

// NewWithSource 以自定义 SecretSource 创建(A5:per-NAS 注册表密钥)。
// 库层丢包(密钥不符/解析失败)统一经 ErrorLog 带 [aaa] 前缀输出,失败路径可 grep。
func NewWithSource(addr string, src radius.SecretSource, h radius.Handler) *Server {
	return &Server{
		srv: &radius.PacketServer{
			Addr:         addr,
			Network:      "udp",
			Handler:      h,
			SecretSource: src,
			ErrorLog:     log.New(os.Stderr, "[aaa] ", log.LstdFlags),
		},
		done: make(chan struct{}),
	}
}

// Start 启动监听,返回错误仅用于启动失败;正常运行时阻塞。
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Shutdown 优雅停止:停止接受新请求并等待在途请求结束。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
