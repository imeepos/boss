package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
	"github.com/ymm-001/boss/internal/pkg/events"
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

	Billing billing.BillingService
	Arrears billing.ArrearsService

	Resource       resource.ResourceService
	ResourceSub    resource.ResourceSubService
	ResourceAssign resource.ResourceAssignService

	Order       order.OrderService
	WorkOrder   order.WorkOrderService
	OrderLedger order.OrderLedgerService
	Channel     order.ChannelService

	Device device.DeviceService
	Alarm  device.AlarmService

	Aaa       aaa.AaaService
	Provision provision.ProvisionService
	QuadLink  quadlink.QuadLinkService
	Asset     asset.AssetService

	Worker       worker.WorkerService
	WorkerLedger worker.WorkerLedgerService
	WorkerFact   worker.WorkerFactService
	WorkerEvent  worker.WorkerEventService

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

// customerLookup 把 customer.CustomerService.Get 适配为 order.CustomerLookup.Exists。
type customerLookup struct{ svc customer.CustomerService }

func (c customerLookup) Exists(ctx context.Context, id int64) (bool, error) {
	_, err := c.svc.Get(ctx, id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, customer.ErrCustomerNotFound) {
		return false, nil
	}
	return false, err
}

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
	ord := order.NewPGStore(pool, customerLookup{svc: cust}, res)
	dev := device.NewPGStore(pool)
	wrk := worker.NewPGStore(pool)
	usr := user.NewPGStore(pool)
	aw := audit.NewAsyncWriter(audit.NewPGWriter(pool), 1024)

	app := &Application{
		User:      usr,
		OrgLedger: usr,

		Customer:       cust,
		Product:        cust,
		CustomerLedger: cust,
		RealName:       cust,

		Billing: bill,
		Arrears: bill,

		Resource:       res,
		ResourceSub:    res,
		ResourceAssign: res,

		Order:       ord,
		WorkOrder:   ord,
		OrderLedger: ord,
		Channel:     ord,

		Device: dev,
		Alarm:  dev,

		Aaa:       aaa.NewPGStore(pool),
		Provision: provision.NewPGStore(pool),
		QuadLink:  quadlink.NewPGStore(pool),
		Asset:     asset.NewPGStore(pool),

		Worker:       wrk,
		WorkerLedger: wrk,
		WorkerFact:   wrk,
		WorkerEvent:  wrk,
	}

	app.Audit = aw

	// W8:Kafka 状态变更链路(brokers 可用即接,否则降级 Noop)。
	var pub events.Publisher = events.Noop{}
	var closePub func()
	if len(cfg.Kafka.Brokers) > 0 {
		kp := events.NewKafkaPublisher(cfg.Kafka.Brokers, cfg.Events.Topic)
		pub = kp
		closePub = func() { _ = kp.Close() }
	}
	app.pubEvents = pub
	app.Automation = NewAutomation(app.Order, pub)

	app.close = func() {
		aw.Close() // 排空审计队列
		if closePub != nil {
			closePub()
		}
		pool.Close()
	}
	return app, nil
}
