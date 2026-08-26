package app

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/aaa"
	aaability "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/ai"
	"github.com/ymm-001/boss/internal/domain/analytics"
	"github.com/ymm-001/boss/internal/domain/apikey"
	"github.com/ymm-001/boss/internal/domain/apprelease"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/backup"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/cms"
	crashdomain "github.com/ymm-001/boss/internal/domain/crash"
	"github.com/ymm-001/boss/internal/domain/cs"
	"github.com/ymm-001/boss/internal/domain/customer"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/domain/gis"
	"github.com/ymm-001/boss/internal/domain/loy"
	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/domain/openplat"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/partner"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/domain/provision"
	pushdomain "github.com/ymm-001/boss/internal/domain/push" // 设备注册表(域侧);通道 Sender 在 pkg/push
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/events"
	"github.com/ymm-001/boss/internal/pkg/hostctl"
	"github.com/ymm-001/boss/internal/pkg/push"
	"github.com/ymm-001/boss/internal/pkg/realid"
	"github.com/ymm-001/boss/internal/pkg/stripe"
)

// Application 持有各域服务的装配结果,是模块化单体依赖绑定的唯一入口。
//
// 依赖倒置(见 docs/ADR-001):域之间只经接口依赖;wiring.go 层把「接口 → 实现」绑定。
// 增量装配(见 docs/architecture-review.md 发现 3.4):每个阶段只构造已实现的域服务。
type Application struct {
	User      user.Service
	OrgLedger user.OrgLedgerService

	Customer       customer.CustomerService
	Product        customer.ProductService
	CustomerLedger customer.CustomerLedgerService
	RealName       customer.RealNameService
	UserData       udcustomer.Service

	// Promotion 营销促销域:券模板/发放/兑换/转赠/缴费抵扣(docs/design/promotion-coupon.md)。
	Promotion promotion.Service

	// Points 忠诚度积分域(最小实现,000104):账本/流水/积分换券。
	Points loy.Service

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
	// CrashLogs 客户端崩溃日志存储(迁移 000136);nil 安全(httpapi 跳过)。
	CrashLogs crashdomain.Store
	// AppRelease 客户端版本发布域(000137):双端 APK 上传/灰度/发布;nil 安全。
	AppRelease *apprelease.Service

	Billing billing.BillingService
	Arrears billing.ArrearsService
	// Dunning 欠费催收域(Q3):逾期标记+欠费清单;停机编排见 RunDunning。
	Dunning billing.DunningService
	Recon   billing.ReconService
	// ReconAuto 自动对账编排(渠道源注册表 ReconSources);admin POST /reconciliations/auto。
	ReconAuto    *billing.AutoReconciler
	ReconSources *billing.ChannelSourceRegistry
	// PayGateway 支付渠道网关注册表(Stripe 卡收单);动态网关,配置驱动(DB/env,60s 热生效)。
	PayGateway *billing.PaymentGatewayRegistry
	// Stripe Stripe 卡收单动态配置;未配置/未启用时发起端点 400、webhook 503 降级。
	Stripe *stripe.Dynamic
	Tax    billing.TaxService
	// TaxGateway 税局网关注册表(CN 数电票/PH BIR eIS);nil=全人工模式(回填票号)。
	TaxGateway *billing.TaxGatewayRegistry

	Resource       resource.ResourceService
	ResourceSub    resource.ResourceSubService
	ResourceAssign resource.ResourceAssignService

	Order           order.OrderService
	WorkOrder       order.WorkOrderService
	CSMetrics       order.CSMetricsReader
	Knowledge       cs.KnowledgeService
	Callbacks       cs.CallbackService
	Tickets         cs.TicketService
	CMS             cms.Service
	ARMetrics       billing.ARMetricsReader
	ARClosure       billing.ARClosureService
	CollectionQueue billing.CollectionQueueService
	OrderLedger     order.OrderLedgerService
	Channel         order.ChannelService

	Device    device.DeviceService
	Alarm     device.AlarmService
	Geo       geo.GeoService
	Gis       gis.GISService
	ODN       odn.ODNService
	Analytics analytics.AnalyticsService
	Report    *report.ReportService

	// Metric 指标目录与数据质量规则(S5 基础):指标 key/定义/公式/负责人/版本/血缘占位;
	// 质量异常经 ScanQuality 发现后投递到 CompTask 补偿任务中心。
	Metric     metric.Service
	ETL        metric.ETLService
	ETLScanner ETLScanner

	Aaa         aaa.AaaService
	Provision   provision.ProvisionService
	QuadLink    quadlink.QuadLinkService
	Asset       asset.AssetService
	APIKey      apikey.Service
	OpenPlat    openplat.Service            // 开放平台(Q4):外部应用凭证/签名/配额/Webhook 订阅
	OpenWebhook *openplat.WebhookDispatcher // Webhook 投递器(outbox/退避重试,迁移 000124)
	AI          ai.Service
	// Partner 招商引资/合作入驻域(迁移 000098)。
	Partner           partner.Service
	PartnerCommission partner.CommissionLedgerService
	PartnerAudit      partner.AuditReportService
	PartnerOrderRisk  partner.OrderRiskService
	PartnerRegion     partner.RegionScopeService

	// Notify 后台提醒中心(admin 通知+待办,迁移 000090)。
	Notify notify.Service

	// CompTask 统一补偿任务/责任队列(S2):失败来源汇聚 + 领取/转派/重试/回放/关闭/审计。
	CompTask *report.CompTaskService

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

	Worker         worker.WorkerService
	WorkerTeam     worker.TeamService
	WorkerLedger   worker.WorkerLedgerService
	WorkerFact     worker.WorkerFactService
	WorkerEvent    worker.WorkerEventService
	WorkerNotice   worker.WorkerNoticeService
	WorkerLocation worker.LocationService

	// 师傅注册 / 审核 / 实名认证 子域(迁移 000050)。
	WorkerOnboarding worker.OnboardingService
	WorkerRealName   worker.RealNameService

	Audit audit.Writer // 关键操作审计(异步写,见 pkg/audit)

	// Pool PG 连接池(指标采集用,pkg/server 注册 gauge;只读快照不接管生命周期)。
	Pool *pgxpool.Pool

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
