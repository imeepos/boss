package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/analytics"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/backup"
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
	"github.com/ymm-001/boss/internal/pkg/events"
	"github.com/ymm-001/boss/internal/pkg/hostctl"
	"github.com/ymm-001/boss/internal/pkg/push"
	"github.com/ymm-001/boss/internal/pkg/realid"
	"github.com/ymm-001/boss/internal/pkg/sms"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// Application 持有各域服务的装配结果,是模块化单体依赖绑定的唯一入口。
//
// 依赖倒置(见 docs/ADR-001):域之间只经接口依赖;本层把「接口 → 实现」绑定。
// 增量装配(见 docs/architecture-review.md 发现 3.4):每个阶段只构造已实现的域服务。
type Application struct {
	User      user.Service
	OrgLedger user.OrgLedgerService

	Customer       customer.CustomerService
	Product        customer.ProductService
	CustomerLedger customer.CustomerLedgerService
	RealName       customer.RealNameService
	UserData       udcustomer.Service

	// Portal 用户端/师傅端门户状态(验证码/账号/偏好/消息/钱包/单号)。
	Portal portal.Service

	// 客户注册 / 审核 / 实名认证 子域(迁移 000051)。
	CustomerOnboarding customer.OnboardingService
	CustomerRealName   customer.RealNameService
	// RealID 实名二要素自动核验通道(阿里云实人认证 Id2MetaVerify);nil=未配置,提交落 PENDING 人工核验。
	RealID realid.Verifier
	// Push 移动端推送通道(极光 JPush 聚合);凭据缺失时为日志通道。
	Push push.Sender
	// PushDevices 推送设备注册表(迁移 000095);App 上报 RegistrationID,发送链路按主体反查。
	PushDevices pushdomain.DevicesService
	// PushNotifier 定向通知器(派单→师傅手机);nil 安全(httpapi 跳过)。
	PushNotifier pushdomain.WorkerNotifier

	Billing billing.BillingService
	Arrears billing.ArrearsService
	Recon   billing.ReconService
	// ReconAuto 自动对账编排(渠道源注册表 ReconSources);admin POST /reconciliations/auto。
	ReconAuto    *billing.AutoReconciler
	ReconSources *billing.ChannelSourceRegistry
	// PayGateway 支付渠道网关注册表(Stripe 卡收单);nil/未注册=模拟直落账(现状)。
	PayGateway *billing.PaymentGatewayRegistry
	// StripeWebhook Stripe 回调验签上下文;密钥未配置时 webhook 端点直接 503。
	StripeWebhook stripe.Webhook
	Tax           billing.TaxService
	// TaxGateway 税局网关注册表(CN 数电票/PH BIR eIS);nil=全人工模式(回填票号)。
	TaxGateway *billing.TaxGatewayRegistry

	Resource       resource.ResourceService
	ResourceSub    resource.ResourceSubService
	ResourceAssign resource.ResourceAssignService

	Order       order.OrderService
	WorkOrder   order.WorkOrderService
	OrderLedger order.OrderLedgerService
	Channel     order.ChannelService

	Device    device.DeviceService
	Alarm     device.AlarmService
	Geo       geo.GeoService
	Gis       gis.GISService
	ODN       odn.ODNService
	Analytics analytics.AnalyticsService
	Report    *report.ReportService

	Aaa       aaa.AaaService
	Provision provision.ProvisionService
	QuadLink  quadlink.QuadLinkService
	Asset     asset.AssetService
	APIKey    apikey.Service
	AI        ai.Service
	// Notify 后台提醒中心(admin 通知+待办,迁移 000090)。
	Notify notify.Service

	// Backup 数据备份迁移(导出 gzip JSONL 归档 + ON CONFLICT 追加恢复,迁移 000095)。
	Backup *backup.Service

	// Attachment 附件上传(三端共用,对象入 MinIO,元数据入 attachments 表)。
	Attachment *attachment.Service

	// HostCtl 宿主机 sidecar 客户端(轮换 MinIO 密钥等特权操作);nil=未配置。
	HostCtl *hostctl.Client

	// AaaAuth 授权查询(授权器,权威状态=lo_accounts);Cdr 话单投递(PG 落库 + Kafka 双写)。
	// gRPC aaa/v1 GetAuthorization/EmitCDR 依赖,债务偿还:契约服务可在 cmd/server 内直连。
	AaaAuth aaa.Authorizer
	Cdr     aaability.Emitter

	Worker       worker.WorkerService
	WorkerLedger worker.WorkerLedgerService
	WorkerFact   worker.WorkerFactService
	WorkerEvent  worker.WorkerEventService
	WorkerNotice worker.WorkerNoticeService

	// 师傅注册 / 审核 / 实名认证 子域(迁移 000050)。
	WorkerOnboarding worker.OnboardingService
	WorkerRealName   worker.RealNameService

	Audit audit.Writer // 关键操作审计(异步写,见 pkg/audit)

	// Automation W8 环节自动编排(6/7/10/11 自动);事件经 Kafka 状态变更链路发布。
	Automation *Automation
	// pubEvents 事件发布器(Kafka;未配置时 Noop);Close 时释放连接。
	pubEvents events.Publisher

	// close 释放资源钩子(异步审计 writer + PG 池),由 cmd 层在优雅退出时调用。
	close func()
}

// Close 释放装配持有的资源(异步审计排空 + 连接池关闭);幂等。
func (a *Application) Close() {
	if a.close != nil {
		a.close()
	}
}

// ErrNotImplemented 域尚未接入装配时返回,便于调用方降级/提示。
var ErrNotImplemented = errors.New("app: domain service not wired yet")

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
