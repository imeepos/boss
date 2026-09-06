package app

// Aaa + 平台级域服务装配:AAA 鉴权 / 资源配置下发 / 四码合一 / 资产 / API key / AI / 后台通知 / 推送设备表 + 定向通知器。
// 与 Application struct 内字段顺序一一对应(平移自 wiring.go,零行为变更)。

import (
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/aaa/credential"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/asset"
	crashdomain "github.com/ymm-001/boss/internal/domain/crash"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/openplat"
	"github.com/ymm-001/boss/internal/domain/procurement"
	"github.com/ymm-001/boss/internal/domain/provision"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/push"
)

// newAAAStoreWithCred 构造 AAA store 并注入凭据编解码器(admin 重置 LOID 密码的
// 密文落库依赖;密钥 BOSS_AAA_CRED_KEY/FILE 显式注入,空则从 Secret 派生并留 ALERT)。
// server 与 cmd/aaa 必须同一密钥材料,否则跨进程密文互不可解。
func newAAAStoreWithCred(pool *pgxpool.Pool, cfg *config.Config) (*aaa.PGStore, error) {
	codec, derived, err := credential.NewResolved(cfg.AAA.CredKey, cfg.AAA.Secret)
	if err != nil {
		return nil, fmt.Errorf("aaa: credential codec: %w", err)
	}
	if derived {
		log.Println("[aaa] CRED KEY ALERT: BOSS_AAA_CRED_KEY 未设置,使用 Secret 派生密钥(生产必须显式配置)")
	}
	return aaa.NewPGStore(pool).WithCredentialCodec(codec), nil
}

func wireAAAInfra(app *Application, pool *pgxpool.Pool, aaastore *aaa.PGStore, pushSender push.Sender, provStore *provision.PGStore) {
	app.Aaa = aaastore
	app.Provision = provStore
	assetStore := asset.NewPGStore(pool)
	app.Asset = assetStore
	qlStore := quadlink.NewPGStore(pool)
	qlStore.UseAssetSink(assetStore) // P1-T1:装机/拆机资产联动(同库强一致,adopted 2026-09-06-asset-tag-p1-wave)
	app.QuadLink = qlStore
	app.APIKey = apikey.NewPGStore(pool)
	openstore := openplat.NewPGStore(pool)
	app.OpenPlat = openstore
	// Webhook 投递器:同一 store(outbox 读写)+ 默认 HTTP 客户端,后台循环驱动。
	app.OpenWebhook = openplat.NewWebhookDispatcher(openstore, openplat.NewHTTPPoster())
	app.AI = ai.NewService(ai.NewPGStore(pool))
	app.Notify = notify.NewPGStore(pool)
	// Procurement 采购-库存域(增量挂靠,迁移 000163)。
	app.Procurement = procurement.NewPGStore(pool)

	app.CrashLogs = crashdomain.NewPGStore(pool)

	pushdevices := pushdomain.NewDevicesPGStore(pool)
	app.PushDevices = pushdevices
	// 派单等业务事件的定向通知:设备反查 → 通道发送 → push_records 留痕(尽力而为)。
	app.PushNotifier = pushdomain.NewNotifier(pushSender, pushdevices, pushdomain.NewRecordsPGStore(pool))
}
