package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/aaa/radius"
	"github.com/ymm-001/boss/internal/domain/apprelease"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/cms"
	"github.com/ymm-001/boss/internal/domain/cs"
	"github.com/ymm-001/boss/internal/domain/customer"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/internal/domain/monthly"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/partner"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/domain/provision"
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
	partnerSvc := partner.NewPGStore(pool)
	bill := billing.NewPGStore(pool)
	promo := promotion.NewPGStore(pool)
	// 券抵扣与缴费同事务:billing 定义注入点,promotion 提供实现(跨域禁实现依赖)。
	bill.SetCouponDeductor(promo.DeductForPayment)
	// 积分换券:LOY 持账本,PROMO 出券;价查/发券经函数注入(补偿模式,见 adopted note)。
	points := loy.NewPGStore(pool, promo.PointsPrice,
		func(ctx context.Context, tpl, cid int64) (string, error) {
			return promo.IssueToCustomer(ctx, tpl, cid, promotion.SourceLoyalty)
		})
	res := resource.NewPGStore(pool)
	qlStore := quadlink.NewPGStore(pool)
	dev := device.NewPGStore(pool)
	wrk := worker.NewPGStore(pool)
	usr := user.NewPGStore(pool)
	aaastore, err := newAAAStoreWithCred(pool, cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	// 审计必须与业务请求同步落库，避免进程崩溃或队列满时丢失关键操作留痕。
	aw := audit.NewPGWriter(pool)
	// E14:预建当月起 2 个月的审计分区(见 wiring_events.go)。
	if err := ensureAuditPartitions(ctx, pool); err != nil {
		return nil, err
	}
	// 推送通道:Notifier/自检共用,设备表构造见 wiring_aaa_infra.go。
	pushSender := push.NewDynamic(pushConfigResolver(usr, cfg))
	// 验证码短信通道:配置源=biz_params(后台短信配置页,60s 热生效),env 凭据兜底,
	// 凭据齐备走阿里云国际短信(+86/+60 统一),否则降级日志通道(仅开发)。
	portalSvc := portal.NewPGStoreWithSender(pool, sms.NewRouter(
		sms.NewDynamic(smsConfigResolver(usr, cfg)),
		nil, // 后续中国国内报备通道就绪后注入 ByRegion["86"]。
		[]string{"86", "60"},
	))

	// 环节4 预付费当场收款(adopted note 2026-08-22):依赖 portalSvc,故在 portal 之后构造;
	// 赠送阶梯经 promotion 命中(000104:buy_months/gift_months 快照)。
	provStore := provision.NewPGStore(pool)
	ord := order.NewPGStore(pool, customerLookup{svc: cust}, res, portReserver{svc: res},
		quadLinkPrebinder{svc: qlStore},
		prepaidCollector{bill: bill, portal: portalSvc, promo: promo, points: points},
		partnerSvc, usr, userProfileCreator{svc: aaastore},
		provisionTaskCreator{svc: provStore})

	// 阶段9:经营分析后端选择(pg 派生聚合 | starrocks OLAP 宽表,见 wiring_events.go)。
	anaStore, closeOLAP, err := selectAnalytics(ctx, pool, cfg)
	if err != nil {
		return nil, err
	}

	app := &Application{
		Pool:      pool,
		User:      usr,
		OrgLedger: usr,

		Customer:       cust,
		Product:        cust,
		CustomerLedger: cust,
		RealName:       cust,
		UserData:       udcustomer.NewPGStore(pool),
		Promotion:      promo,
		Points:         points,
		Portal:         portalSvc,

		CustomerOnboarding: cust,
		Partner:            partnerSvc,
		PartnerCommission:  partnerSvc,
		PartnerAudit:       partnerSvc,
		PartnerOrderRisk:   partnerSvc,
		PartnerRegion:      partnerSvc,
		CustomerRealName:   cust,
		// 实名二要素通道:biz_params(realid.*)优先/env 兜底,60s 热生效;未配置落 PENDING 人工核验。
		RealID: realid.NewDynamic(realidConfigResolver(usr, cfg)),

		// 移动端推送通道:biz_params(push.*)优先/env 兜底,60s 热生效;凭据缺失降级日志通道。
		Push: pushSender,

		Billing: bill,
		Arrears: bill,
		Dunning: bill,
		Recon:   bill,
		Tax:     bill,

		// 自动对账编排:渠道源注册表,真实渠道凭据到货后在此 Register。
		ReconSources: billing.NewChannelSourceRegistry(),
		ReconAuto:    nil, // New() 尾部装配(需要 Recon 就绪)

		Resource:       res,
		ResourceSub:    res,
		ResourceAssign: res,

		ResourceCapacity:  res,
		CapacityAlarmSink: newCapacityAlarmSink(dev),

		Order:           ord,
		WorkOrder:       ord,
		CSMetrics:       ord,
		Knowledge:       cs.NewPGKnowledgeStore(pool),
		Callbacks:       cs.NewPGKnowledgeStore(pool),
		Tickets:         cs.NewPGKnowledgeStore(pool),
		CMS:             cms.NewPGStore(pool),
		ARMetrics:       bill,
		ARClosure:       bill,
		CollectionQueue: bill,
		OrderLedger:     ord,
		Channel:         ord,

		Device:    dev,
		Analytics: anaStore,
		Monthly:   monthly.NewPGStore(pool),
		Alarm:     dev,

		// Backup 数据备份迁移(运维工具);归档目录 env BOSS_BACKUP_DIR,默认 data/backups。
		Backup: newBackupService(pool),

		Attachment: &attachment.Service{
			St: attachment.NewPGStore(pool), Obj: attachment.NewMinIOStorage(),
			Conf: minioFallback(cfg),
		},

		// AppRelease 客户端版本发布(000137):APK 对象复用 attachment 的 MinIO 抽象。
		AppRelease: &apprelease.Service{
			St: apprelease.NewPGStore(pool), Obj: attachment.NewMinIOStorage(),
			Conf: minioFallback(cfg),
		},

		// License 系统级授权门禁(release-platform 离线授权)。
		// Enabled=false 时保持 nil(完全放行);Enabled 但缺公钥视为配置错误,启动即失败。
		License: wireLicenseService(cfg),

		Worker:         wrk,
		WorkerTeam:     wrk,
		WorkerLedger:   wrk,
		WorkerFact:     wrk,
		WorkerEvent:    wrk,
		WorkerNotice:   wrk,
		WorkerLocation: wrk,

		WorkerOnboarding: wrk,
		WorkerRealName:   wrk,
	}

	wireGeoServices(app, pool)
	wireAAAInfra(app, pool, aaastore, pushSender, provStore)
	// 订单环节推进广播到开放平台 Webhook(000125 outbox;尽力而为,失败不影响推进)。
	ord.SetStageNotifier(app.OpenWebhook)

	app.Audit = aw
	app.Attachment.Resolve = minioConfigResolver(app.User, app.Attachment.Conf)
	app.AppRelease.Resolve = app.Attachment.Resolve // 同源密钥热轮换
	if cfg.HostCtl.URL != "" && cfg.HostCtl.HMACKey != "" {
		app.HostCtl = hostctl.New(cfg.HostCtl.URL, cfg.HostCtl.HMACKey)
	}
	wireStripe(app, usr) // 卡收单通道:配置驱动(DB biz_params,60s 热生效,见 wiring_stripe.go)

	// 阶段9:经营分析 + 自动报告。
	reportStore := report.NewPGStore(pool)
	app.Report = &report.ReportService{Ana: app.Analytics, St: reportStore}
	app.CompTask = report.NewCompTaskService(reportStore)
	app.Metric = metric.NewPGStore(pool)
	app.ETL = metric.NewPGStore(pool)
	app.ETLScanner = NewETLScanner(app.ETL, app.CompTask)

	// 债务偿还:gRPC aaa/v1 依赖——授权器 + 话单投递 + W8 事件链(见 wiring_events.go)。
	app.AaaAuth = aaa.NewPGAuthorizer(pool)
	em := wireEmitters(aaastore, cfg)
	app.Cdr = em.cdr
	app.pubEvents = em.pub

	// AAA-A2 在线会话控制:CoA 下发器(NAS 3799 可配)+ 有限次重试 + 僵尸清理。
	app.SessCtl = aaa.NewSessionControlService(aaastore,
		radius.NewCoAClient([]byte(cfg.AAA.Secret), cfg.AAA.CoAPort, 5*time.Second), cfg.AAA.OfflineRetryMax)
	app.Automation = NewAutomation(app.Order, em.pub)
	app.ReconAuto = &billing.AutoReconciler{Recon: app.Recon, Sources: app.ReconSources}

	stopPatrol := startPatrolLoop(app)
	stopReserveTimeout := startReserveTimeoutLoop(app)
	stopCdrComp := startCdrCompensationLoop(aaastore, em.cdrRT, app.Notify)
	stopOfflineRetry := startAAAOfflineRetryLoop(app.SessCtl)
	stopZombieReap := startZombieReapLoop(app.SessCtl, cfg.AAA.ZombieAfter)
	stopDailyRecon := startDailyReconLoop(app)
	stopOSSAudit := startOSSAuditLoop(app)
	stopPointsExpire := startPointsExpireLoop(points)
	stopWebhookDelivery := startWebhookDeliveryLoop(app.OpenWebhook)
	stopETLOverdue := startETLOverdueLoop(app)
	stopETLProjection := startETLProjectionLoop(app)
	stopStripeGuard := startStripeWebhookGuard(app)
	app.close = func() {
		stopPatrol()          // 巡检循环
		stopReserveTimeout()  // 预占超时释放循环(Q2)
		stopCdrComp()         // 话单补偿循环(Q2)
		stopOfflineRetry()    // Disconnect 重试循环(AAA-A2)
		stopZombieReap()      // 僵尸会话清理循环(AAA-A2)
		stopDailyRecon()      // 每日数据对账循环(Q2)
		stopOSSAudit()        // 资源台账稽核每日快照循环(P5-W3)
		stopPointsExpire()    // 积分过期清算循环(2028 Q2)
		stopWebhookDelivery() // Webhook 投递循环(Q4 开放平台 M2)
		stopETLOverdue()      // ETL overdue 自动派单
		stopETLProjection()   // ETL 真实投影执行器
		stopStripeGuard()     // Stripe webhook endpoint 自愈/告警循环
		// PGWriter 为同步写入器，无需排空进程内队列。
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
