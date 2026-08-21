package app

import (
	"context"
	"fmt"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/gis"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/provision"
	pushdomain "github.com/ymm-001/boss/internal/domain/push" // 设备注册表(域侧);通道 Sender 在 pkg/push
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
	"github.com/ymm-001/boss/internal/pkg/hostctl"
	"github.com/ymm-001/boss/internal/pkg/push"
	"github.com/ymm-001/boss/internal/pkg/realid"
	"github.com/ymm-001/boss/internal/pkg/sms"
)

// New 装配依赖:打开 PG → 应用迁移 → 构造各域 PGStore。
// migrationsDir 为 migrations/*.up.sql 所在目录(通常相对工作目录为 "migrations")。
func New(ctx context.Context, cfg *config.Config, migrationsDir string) (*Application, error) {
	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("app: open database: %w", err)
	}
	if err := database.Migrate(ctx, pool, migrationsDir); err != nil {
		pool.Close()
		return nil, fmt.Errorf("app: migrate: %w", err)
	}

	cust := customer.NewPGStore(pool)
	bill := billing.NewPGStore(pool)
	res := resource.NewPGStore(pool)
	qlStore := quadlink.NewPGStore(pool)
	ord := order.NewPGStore(pool, customerLookup{svc: cust}, res, portReserver{svc: res}, quadLinkPrebinder{svc: qlStore})
	dev := device.NewPGStore(pool)
	wrk := worker.NewPGStore(pool)
	usr := user.NewPGStore(pool)
	aaastore := aaa.NewPGStore(pool)
	akstore := apikey.NewPGStore(pool)
	aisvc := ai.NewService(ai.NewPGStore(pool))
	aw := audit.NewAsyncWriter(audit.NewPGWriter(pool), 1024)
	// E14:预建当月起 2 个月的审计分区(见 wiring_events.go)。
	if err := ensureAuditPartitions(ctx, pool); err != nil {
		return nil, err
	}
	// 推送通道与设备注册表:通道供 Notifier/自检共用;设备表供上报与反查共用。
	pushSender := push.NewDynamic(pushConfigResolver(usr, cfg))
	pushdevices := pushdomain.NewDevicesPGStore(pool)
	// 验证码短信通道:配置源=biz_params(后台短信配置页,60s 热生效),env 凭据兜底,
	// 凭据齐备走阿里云国际短信(+86/+60 统一),否则降级日志通道(仅开发)。
	portalSvc := portal.NewPGStoreWithSender(pool, sms.NewRouter(
		sms.NewDynamic(smsConfigResolver(usr, cfg)),
		nil, // 后续中国国内报备通道就绪后注入 ByRegion["86"]。
		[]string{"86", "60"},
	))

	// 阶段9:经营分析后端选择(pg 派生聚合 | starrocks OLAP 宽表,见 wiring_events.go)。
	anaStore, closeOLAP, err := selectAnalytics(ctx, pool, cfg)
	if err != nil {
		return nil, err
	}

	app := &Application{
		User:      usr,
		OrgLedger: usr,

		Customer:       cust,
		Product:        cust,
		CustomerLedger: cust,
		RealName:       cust,
		UserData:       udcustomer.NewPGStore(pool),
		Portal:         portalSvc,

		CustomerOnboarding: cust,
		CustomerRealName:   cust,
		// 实名二要素通道:biz_params(realid.*)优先/env 兜底,60s 热生效;未配置落 PENDING 人工核验。
		RealID: realid.NewDynamic(realidConfigResolver(usr, cfg)),

		// 移动端推送通道:biz_params(push.*)优先/env 兜底,60s 热生效;凭据缺失降级日志通道。
		Push: pushSender,

		Billing: bill,
		Arrears: bill,
		Recon:   bill,
		Tax:     bill,

		// 自动对账编排:渠道源注册表,真实渠道凭据到货后在此 Register。
		ReconSources: billing.NewChannelSourceRegistry(),
		ReconAuto:    nil, // New() 尾部装配(需要 Recon 就绪)

		Resource:       res,
		ResourceSub:    res,
		ResourceAssign: res,

		Order:       ord,
		WorkOrder:   ord,
		OrderLedger: ord,
		Channel:     ord,

		Device:    dev,
		Geo:       geo.NewPGStore(pool),
		Gis:       gis.NewPGStore(pool),
		ODN:       odn.NewPGStore(pool),
		Analytics: anaStore,
		Alarm:     dev,

		Aaa:         aaastore,
		Provision:   provision.NewPGStore(pool),
		QuadLink:    qlStore,
		Asset:       asset.NewPGStore(pool),
		APIKey:      akstore,
		AI:          aisvc,
		Notify:      notify.NewPGStore(pool),
		PushDevices: pushdevices,
		// 派单等业务事件的定向通知:设备反查 → 通道发送 → push_records 留痕(尽力而为)。
		PushNotifier: pushdomain.NewNotifier(pushSender, pushdevices, pushdomain.NewRecordsPGStore(pool)),

		// Backup 数据备份迁移(运维工具);归档目录 env BOSS_BACKUP_DIR,默认 data/backups。
		Backup: newBackupService(pool),

		Attachment: &attachment.Service{
			St: attachment.NewPGStore(pool), Obj: attachment.NewMinIOStorage(),
			Conf: minioFallback(cfg),
		},

		Worker:       wrk,
		WorkerLedger: wrk,
		WorkerFact:   wrk,
		WorkerEvent:  wrk,
		WorkerNotice: wrk,

		WorkerOnboarding: wrk,
		WorkerRealName:   wrk,
	}

	app.Audit = aw
	app.Attachment.Resolve = minioConfigResolver(app.User, app.Attachment.Conf)
	if cfg.HostCtl.URL != "" && cfg.HostCtl.HMACKey != "" {
		app.HostCtl = hostctl.New(cfg.HostCtl.URL, cfg.HostCtl.HMACKey)
	}
	wireStripe(app, cfg) // 卡收单通道:密钥齐备才注册(见 wiring_stripe.go)

	// 阶段9:经营分析 + 自动报告。
	app.Report = &report.ReportService{Ana: app.Analytics, St: report.NewPGStore(pool)}

	// 债务偿还:gRPC aaa/v1 依赖——授权器 + 话单投递 + W8 事件链(见 wiring_events.go)。
	app.AaaAuth = aaa.NewPGAuthorizer(pool)
	em := wireEmitters(aaastore, cfg)
	app.Cdr = em.cdr
	app.pubEvents = em.pub
	app.Automation = NewAutomation(app.Order, em.pub)
	app.ReconAuto = &billing.AutoReconciler{Recon: app.Recon, Sources: app.ReconSources}

	stopPatrol := startPatrolLoop(app)
	app.close = func() {
		stopPatrol() // 巡检循环
		aw.Close()   // 排空审计队列
		if em.closeCdr != nil {
			em.closeCdr()
		}
		if em.closePub != nil {
			em.closePub()
		}
		if closeOLAP != nil {
			closeOLAP()
		}
		pool.Close()
	}
	return app, nil
}
