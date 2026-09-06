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
	// AAA-A5:per-NAS 注册表密钥校验(SecretSource)+ 厂商 VSA 限速下发(Handler)。
	vsaSpec, err := aaa.BuildVSASpec(aaa.VSAConfig{HuaweiSpec: cfg.AAA.VSAHuawei, ZTESpec: cfg.AAA.VSAZTE})
	if err != nil {
		log.Fatalf("[aaa] vsa spec: %v", err)
	}
	handler := &radius.Handler{
		Auth: auth, CDR: buildEmitter(cfg, store), Log: store,
		Sessions: store, Gate: store, SessionLimit: cfg.AAA.SessionLimit,
		Nas: store, VSA: vsaSpec,
	}

	source := &radius.RegistrySecretSource{
		Registry: store, Global: []byte(cfg.AAA.Secret), Compat: cfg.AAA.GlobalSecretCompat,
	}
	authSrv := radius.NewWithSource(cfg.AAA.AuthAddr, source, handler)
	acctSrv := radius.NewWithSource(cfg.AAA.AcctAddr, source, handler)

	go serve(authSrv, "auth")
	go serve(acctSrv, "acct")

	<-ctx.Done()
	log.Println("aaa: shutting down")
}

// buildCodec 凭据编解码器:CRED_KEY 显式优先,空则从 Secret 派生(域内 BuildCodec 统一口径,
// cmd/aaa 与 app 装配共用;派生为开发兜底,生产必须显式设置独立密钥)。
func buildCodec(cfg *config.Config) *credential.Codec {
	codec, err := aaa.BuildCodec(cfg.AAA.CredKey, cfg.AAA.Secret)
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
