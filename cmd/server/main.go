// server 业务单体入口:阶段1-6 全部业务域 + 阶段7 接口预留。
package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
	bosotel "github.com/ymm-001/boss/internal/pkg/otel"
	"github.com/ymm-001/boss/internal/pkg/server"
)

func main() {
	cfg := config.Load()

	// OTel→Jaeger(OTLP gRPC,如 192.168.0.102:16831);端点空则降级 noop。
	shutdownOtel, err := bosotel.Setup(context.Background(), bosotel.Config{
		Endpoint: cfg.Observability.JaegerOTLP, Service: "boss-server",
	})
	if err != nil {
		log.Fatalf("otel setup: %v", err)
	}
	defer func() { _ = shutdownOtel(context.Background()) }()

	a, err := app.New(context.Background(), cfg, "migrations")
	if err != nil {
		log.Fatalf("wiring: %v", err)
	}

	mgr := auth.NewManager(cfg.JWT.Secret, cfg.JWT.TTL)
	r := server.New(server.Config{HTTPAddr: cfg.Server.HTTPAddr})
	app.RegisterRoutes(r, a, mgr)

	// 债务偿还:同进程起 gRPC 服务间契约(quadlink/aaa/device/provision v1)。
	grpcSrv := grpc.NewServer()
	app.RegisterGRPC(grpcSrv, a)
	go serveGRPC(cfg.Server.GRPCAddr, grpcSrv)

	if err := server.Run(r, cfg.Server.HTTPAddr); err != nil {
		log.Fatalf("server: %v", err)
	}
	// 优雅退出后,先停 gRPC 再排空异步审计队列并关闭连接池。
	grpcSrv.GracefulStop()
	a.Close()
}

// serveGRPC 在独立 goroutine 中启动 gRPC 监听;启动失败即退出。
func serveGRPC(addr string, s *grpc.Server) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	log.Printf("boss-server grpc listening on %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
