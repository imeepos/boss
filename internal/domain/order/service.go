package order

import (
	"context"
	"errors"
)

// ErrOrderNotFound 订单不存在。
var ErrOrderNotFound = errors.New("order: not found")

// ErrIllegalTransition 非法状态流转(经 statemachine 判定拒绝)。
var ErrIllegalTransition = errors.New("order: illegal transition")

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

	// List 订单列表读模型(listOrders);GetByNo 按订单号寻址(getOrder)。
	List(ctx context.Context, q OrderQuery) ([]OrderListItem, error)
	GetByNo(ctx context.Context, orderNo string) (*Order, error)

	// Track 跟踪(order.html 时间轴):返回订单 + 环节日志。
	Track(ctx context.Context, orderID int64) (*Order, []StageLog, error)

	// ChangeAddress 变更安装地址(用户端 change-address)。
	ChangeAddress(ctx context.Context, orderID, addressID int64) error

	// SaveRating 落订单评价;RatingExists 判定已评价。
	SaveRating(ctx context.Context, r Rating) error
	RatingExists(ctx context.Context, orderNo string) (bool, error)
}
