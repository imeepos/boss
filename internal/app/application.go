package app

import (
	"errors"

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
	"github.com/ymm-001/boss/internal/domain/loy"
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

	Billing billing.BillingService
	Arrears billing.ArrearsService
	// Dunning 欠费催收域(Q3):逾期标记+欠费清单;停机编排见 RunDunning。
	Dunning billing.DunningService
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
	OpenPlat  openplat.Service // 开放平台(Q4):外部应用凭证/签名/配额/Webhook 订阅
	AI        ai.Service
	// Partner 招商引资/合作入驻域(迁移 000098)。
	Partner           partner.Service
	PartnerCommission partner.CommissionLedgerService
	PartnerAudit      partner.AuditReportService
	PartnerOrderRisk  partner.OrderRiskService
