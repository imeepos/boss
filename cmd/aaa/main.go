// aaa 服务入口:自研 Go RADIUS(认证 1812 / 计费 1813),阶段7 独立部署。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/aaa/radius"
	"github.com/ymm-001/boss/internal/pkg/config"
)

func main() {
	cfg := config.Load()

	// 阶段7骨架:内存 Authorizer 验证协议链路;落地替换为 DB+Redis 实现。
	auth := aaa.NewMemoryAuthorizer([]aaa.Profile{{
		LOID:       "demo",
		Status:     aaa.StatusActive,
		Bandwidth:  "100M/50M",
		SessionTTL: cfg.AAA.AuthTTL,
	}})

	handler := &radius.Handler{Auth: auth} // CDR 阶段7接入 Kafka Emitter
	authSrv := radius.New(cfg.AAA.AuthAddr, []byte(cfg.AAA.Secret), handler)
	acctSrv := radius.New(cfg.AAA.AcctAddr, []byte(cfg.AAA.Secret), handler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go serve(authSrv, "auth")
	go serve(acctSrv, "acct")

	<-ctx.Done()
	log.Println("aaa: shutting down")
}

func serve(s *radius.Server, name string) {
	if err := s.Start(); err != nil {
		log.Printf("aaa %s server: %v", name, err)
		os.Exit(1)
	}
}
