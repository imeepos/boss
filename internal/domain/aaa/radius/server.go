package radius

import (
	"context"

	"layeh.com/radius"
)

// Server 封装 radius.PacketServer 生命周期与优雅退出。
type Server struct {
	srv  *radius.PacketServer
	done chan struct{}
}

// New 创建 RADIUS 服务;addr 形如 ":1812"(认证)或 ":1813"(计费)。
func New(addr string, secret []byte, h radius.Handler) *Server {
	return &Server{
		srv: &radius.PacketServer{
			Addr:         addr,
			Network:      "udp",
			Handler:      h,
			SecretSource: radius.StaticSecretSource(secret),
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
