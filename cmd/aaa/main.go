// aaa 服务入口:自研 Go RADIUS(认证 1812 / 计费 1813),阶段7 独立部署。
// W6 落地:PG 授权器(lo_accounts 权威状态) + 话单双写(PG 落库 + Kafka 实时)。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaabilling "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/aaa/radius"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("aaa: open database: %v", err)
	}
	defer pool.Close()

	store := aaa.NewPGStore(pool)
	auth := aaa.NewPGAuthorizer(pool) // 停复机即时生效:状态权威源=lo_accounts

	handler := &radius.Handler{Auth: auth, CDR: buildEmitter(cfg, store)}

	authSrv := radius.New(cfg.AAA.AuthAddr, []byte(cfg.AAA.Secret), handler)
	acctSrv := radius.New(cfg.AAA.AcctAddr, []byte(cfg.AAA.Secret), handler)

	go serve(authSrv, "auth")
	go serve(acctSrv, "acct")

	<-ctx.Done()
	log.Println("aaa: shutting down")
}

// buildEmitter 话单投递:PG 落库(必选,账务兜底) + Kafka 实时(可用时双写)。
func buildEmitter(cfg *config.Config, store *aaa.PGStore) aaabilling.Emitter {
	pg := aaabilling.NewPGEmitter(store)
	if len(cfg.Kafka.Brokers) == 0 {
		return pg
	}
	return &aaabilling.FanoutEmitter{
		Realtime: aaabilling.NewKafkaEmitter(cfg.Kafka.Brokers, cfg.AAA.CDRTopic),
		Store:    pg,
	}
}

func serve(s *radius.Server, name string) {
	if err := s.Start(); err != nil {
		log.Printf("aaa %s server: %v", name, err)
		os.Exit(1)
	}
}
