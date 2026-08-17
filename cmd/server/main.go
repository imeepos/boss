// server 业务单体入口:阶段1-6 全部业务域 + 阶段7 接口预留。
package main

import (
	"context"
	"log"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/server"
)

func main() {
	cfg := config.Load()
	a, err := app.New(context.Background(), cfg, "migrations")
	if err != nil {
		log.Fatalf("wiring: %v", err)
	}

	mgr := auth.NewManager(cfg.JWT.Secret, cfg.JWT.TTL)
	r := server.New(server.Config{HTTPAddr: cfg.Server.HTTPAddr})
	app.RegisterRoutes(r, a, mgr)

	if err := server.Run(r, cfg.Server.HTTPAddr); err != nil {
		log.Fatalf("server: %v", err)
	}
	// 优雅退出后,排空异步审计队列并关闭连接池。
	a.Close()
}
