// aaa 服务入口:自研 Go RADIUS(认证 1812 / 计费 1813),阶段7 独立部署。
// W6 落地:PG 授权器(lo_accounts 权威状态) + 话单双写(PG 落库 + Kafka 实时)。
// A1 落地:凭据校验决策器(PAP/CHAP + 防爆破锁定)。
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaabilling "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/aaa/credential"
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

	codec := buildCodec(cfg)
	store := aaa.NewPGStore(pool).WithCredentialCodec(codec) // 管理端重置密码:密文落库
	auth := aaa.NewCredentialAuthorizer(pool, aaa.CredentialConfig{
		Codec:         codec,
		AllowNoCred:   cfg.AAA.AllowNoCred,
		LockThreshold: cfg.AAA.LockThreshold,
		LockWindow:    cfg.AAA.LockWindow,
	})

	// AAA-A2:计账链路维护在线会话 + 并发会话闸口(上限全局可配,默认 1)。
	handler := &radius.Handler{
		Auth: auth, CDR: buildEmitter(cfg, store), Log: store,
		Sessions: store, Gate: store, SessionLimit: cfg.AAA.SessionLimit,
	}

	authSrv := radius.New(cfg.AAA.AuthAddr, []byte(cfg.AAA.Secret), handler)
	acctSrv := radius.New(cfg.AAA.AcctAddr, []byte(cfg.AAA.Secret), handler)

	go serve(authSrv, "auth")
	go serve(acctSrv, "acct")

	<-ctx.Done()
	log.Println("aaa: shutting down")
}

// buildCodec 凭据编解码器:BOSS_AAA_CRED_KEY 显式配置优先;空则从 AAA.Secret 派生
// (开发兜底;派生时打 ALERT 供运维感知,生产必须显式设置独立密钥)。
func buildCodec(cfg *config.Config) *credential.Codec {
	material := cfg.AAA.CredKey
	if material == "" {
		material = "boss-aaa-cred-key|" + cfg.AAA.Secret
		log.Println("[aaa] CRED KEY ALERT: BOSS_AAA_CRED_KEY 未设置,使用 Secret 派生密钥(生产必须显式配置)")
	}
	codec, err := credential.New(material)
	if err != nil {
		log.Fatalf("[aaa] credential codec: %v", err)
	}
	return codec
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
