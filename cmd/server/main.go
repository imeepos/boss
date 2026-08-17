// server 业务单体入口:阶段1-6 全部业务域 + 阶段7 接口预留。
package main

import (
	"context"
	"log"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/config"
)

func main() {
	cfg := config.Load()
	a, err := app.New(context.Background(), cfg, "migrations")
	if err != nil {
		log.Fatalf("wiring: %v", err)
	}
	_ = a
	log.Println("boss-server: ready")
}
