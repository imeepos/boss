package order

import (
	"context"
	"errors"
)

// ErrOrderNotFound 订单不存在。
var ErrOrderNotFound = errors.New("order: not found")

// ErrIllegalTransition 非法状态流转(经 statemachine 判定拒绝)。
var ErrIllegalTransition = errors.New("order: illegal transition")
var ErrPartnerDailyCap = errors.New("order: partner daily order cap exceeded")
var ErrPartnerCustomerCooldown = errors.New("order: partner customer cooldown")

// 直营下单风控哨兵(阈值经 biz_params risk.direct.* 可调,见 pg_risk_direct.go)。
var ErrDirectPhoneCap = errors.New("order: direct phone order cap exceeded")
var ErrDirectAddressCap = errors.New("order: direct address in-flight cap exceeded")

// ResourceChecker 资源核查跨域依赖口(契约 CT-002)。
// 由 resource 域(阶段4)提供、app 装配层注入;order 域不 import resource 域实现。
type ResourceChecker interface {
	// Check 核查目标地址是否有空闲端口;available=false 时给出可选方案。
	Check(ctx context.Context, addressID int64) (available bool, options []string, err error)
}

// PortReserver 端口预占跨域依赖口(环节3/5)。
// 由 resource 域提供,在目标地址找空闲端口预占给订单;无空闲返回 ErrPortNotAvailable。
type PortReserver interface {
	ReserveFirstAvailable(ctx context.Context, addressID, orderID int64) (portID int64, err error)
}

// PrepaidCollector 预付费当场收款跨域依赖口(环节4)。
// 由 billing+portal+promotion 域提供、app 装配层注入(派 PAY 单号 + 落缴费流水
// + 赠送时长阶梯命中落痕);order 域不 import billing/promotion 域实现。
// amount = months x 月费(调用方算好);返回赠送月数(0=未命中阶梯)。
type PrepaidCollector interface {
	Collect(ctx context.Context, customerID int64, amount float64, offerID int64, months int) (giftMonths int, err error)
}

// QuadLinkBindReq 四码预绑定请求(order 域定义,由 app 装配层映射到 quadlink.QuadLink)。
type QuadLinkBindReq struct {
	AssetID         int64
	CustomerID      int64
	PortID          int64
	AddressID       int64
	LegalEntityID   int64
	LegalEntityName string
	Status          string
}

// QuadLinkPrebinder 四码预绑定跨域依赖口(环节5)。
// 由 quadlink 域提供,创建四码关联行(UNLINKED,资产可为空,扫码时回填)。
type QuadLinkPrebinder interface {
	CreateLink(ctx context.Context, q QuadLinkBindReq) (linkID int64, err error)
}

// UserProfileCreator 创建 LO 认证账号跨域依赖口(环节6)。
// 由 aaa 域提供,幂等创建 lo_accounts;已存在则返回已有账号。
type UserProfileCreator interface {
	CreateLoAccount(ctx context.Context, lo LoidReq) (int64, error)
	GetLoAccountByCustomer(ctx context.Context, customerID int64) (*LoidAccount, error)
	// AlignLoAccount 把已有 LO 账号的生效套餐对齐到订单套餐(改套餐场景,
	// TMF change order 语义:服务配置必须随变更单刷新);返回是否发生更新。
	AlignLoAccount(ctx context.Context, customerID, offerID int64, billingMode string) (bool, error)
}

// LoidReq 创建 LO 认证账号请求(order 域定义,由 app 装配层映射到 aaa.LoAccount)。
type LoidReq struct {
	Loid            string
	CustomerID      int64
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	RegionPath      string
	OfferID         int64
	QosTemplateID   int64
	Status          string
	BillingMode     string
}

// LoidAccount LO 账号查询结果(order 域使用的最小视图)。
type LoidAccount struct {
	ID      int64
	Loid    string
	OfferID int64
}

// ProvisionTaskCreator 创建配置下发任务跨域依赖口(环节7)。
// 由 provision 域提供,幂等创建下发任务(同 task_no 复用,FAILED 重置 PENDING)。
type ProvisionTaskCreator interface {
	CreateTask(ctx context.Context, t ProvisionTask) (int64, error)
}

// ProvisionTemplateFinder 环节7 按订单套餐解析下发模板(跨域依赖口,由 provision 域提供)。
// 修复:此前 PreConfigOLT 直接把 product_offers.id 当 provision_templates.id 传入,
// 两表 ID 空间重叠时静默错配,新套餐无同号模板行则外键违规卡死环节7。
type ProvisionTemplateFinder interface {
	// FindTemplateForOffer 返回套餐应套用的模板 ID:
	// 优先按套餐带宽匹配同法人 ENABLED 模板(content->>'bandwidth'),无命中回退法人默认模板。
	FindTemplateForOffer(ctx context.Context, offerID, legalEntityID int64) (int64, error)
}

// ProvisionTask 下发任务请求(order 域定义,由 app 装配层映射到 provision.Task)。
type ProvisionTask struct {
	TaskNo      string
	OrderID     int64
	StageEvent  string
	LoAccountID int64
	TemplateID  int64
	Status      string
}

// OrderService 订单域服务口(阶段5)。
// 契约:CT-002 订单-资源核查、CT-003 端口预占;环节标识见 terms.md §1;状态/环节正交见 terms.md §3。
type OrderService interface {
	// Submit 下单(环节1):校验客户/产品有效后建单,status=PENDING、stage=1。
	Submit(ctx context.Context, req SubmitReq) (*Order, error)
	// CheckResource 资源核查(环节2):调 ResourceChecker,成功推进 stage 到 2。
	CheckResource(ctx context.Context, orderID int64) error
	// Reserve 端口预占(环节3):状态机 PENDING→RESERVED。
	Reserve(ctx context.Context, orderID int64) error

	// 环节 4~12(阶段5 人工模式;阶段7 自动化共用同一状态机)。
	ChargeContract(ctx context.Context, orderID int64) error
	ApplyTag(ctx context.Context, orderID int64) error
	CreateUserProfile(ctx context.Context, orderID int64) error
	PreConfigOLT(ctx context.Context, orderID int64) error
	DispatchOrder(ctx context.Context, orderID int64) error
	ScanBind(ctx context.Context, orderID int64) error
	ActivateUser(ctx context.Context, orderID int64) error
	NotifyActivation(ctx context.Context, orderID int64) error
	UpdateMap(ctx context.Context, orderID int64) error

	// Cancel 取消(任一未完成状态可取消);Release 端口释放(预占回滚)。
	Cancel(ctx context.Context, orderID int64) error
	Release(ctx context.Context, orderID int64) error
	// RollbackStage 回退至上一完成环节(worker 端 rollback):删最新日志、stage 前移、status 逆向迁移。
	// 返回 before/after 环节供调用方做副作用回执(after==before 即 0 行生效,不得静默 200)。
	RollbackStage(ctx context.Context, orderID int64) (before, after int8, err error)

	// List 订单列表读模型(listOrders);GetByNo 按订单号寻址(getOrder)。
	List(ctx context.Context, q OrderQuery) ([]OrderListItem, error)
	GetByNo(ctx context.Context, orderNo string) (*Order, error)

	// Track 跟踪(order.html 时间轴):返回订单 + 环节日志。
	Track(ctx context.Context, orderID int64) (*Order, []StageLog, error)

	// PrepaidAmount 预付费订单应收(环节4/师傅现场收款同口径):金额=月数×月费,
	// 后付费返回 prepaid=false。2026-09-05 师傅端现场收款落账用。
	PrepaidAmount(ctx context.Context, orderID int64) (amount float64, months int, prepaid bool, err error)

	// ChangeAddress 变更安装地址(用户端 change-address)。
	ChangeAddress(ctx context.Context, orderID, addressID int64) error

	// SaveRating 落订单评价;RatingExists 判定已评价。
	SaveRating(ctx context.Context, r Rating) error
	RatingExists(ctx context.Context, orderNo string) (bool, error)
}
