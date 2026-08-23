package app

// Aaa + 平台级域服务装配:AAA 鉴权 / 资源配置下发 / 四码合一 / 资产 / API key / AI / 后台通知 / 推送设备表 + 定向通知器。
// 与 Application struct 内字段顺序一一对应(平移自 wiring.go,零行为变更)。

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/openplat"
	"github.com/ymm-001/boss/internal/domain/provision"
	pushdomain "github.com/ymm-001/boss/internal/domain/push"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/push"
)

func wireAAAInfra(app *Application, pool *pgxpool.Pool, aaastore *aaa.PGStore, pushSender push.Sender) {
	app.Aaa = aaastore
	app.Provision = provision.NewPGStore(pool)
	app.QuadLink = quadlink.NewPGStore(pool)
	app.Asset = asset.NewPGStore(pool)
	app.APIKey = apikey.NewPGStore(pool)
	openstore := openplat.NewPGStore(pool)
	app.OpenPlat = openstore
	// Webhook 投递器:同一 store(outbox 读写)+ 默认 HTTP 客户端,后台循环驱动。
	app.OpenWebhook = openplat.NewWebhookDispatcher(openstore, openplat.NewHTTPPoster())
	app.AI = ai.NewService(ai.NewPGStore(pool))
	app.Notify = notify.NewPGStore(pool)

	pushdevices := pushdomain.NewDevicesPGStore(pool)
	app.PushDevices = pushdevices
	// 派单等业务事件的定向通知:设备反查 → 通道发送 → push_records 留痕(尽力而为)。
	app.PushNotifier = pushdomain.NewNotifier(pushSender, pushdevices, pushdomain.NewRecordsPGStore(pool))
}
