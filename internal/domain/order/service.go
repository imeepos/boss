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

// OrderService 订单域服务口(阶段5)。
// 契约:CT-002 订单-资源核查、CT-003 端口预占;未收费不派单(CT-004)属环节4,本切片只留接口位。
type OrderService interface {
	// Submit 下单(环节1):校验客户/产品有效后建单,status=PENDING、stage=1。
	Submit(ctx context.Context, req SubmitReq) (*Order, error)
	// CheckResource 资源核查(环节2):调 ResourceChecker,成功推进 stage 到 2。
	CheckResource(ctx context.Context, orderID int64) error
	// Reserve 端口预占(环节3):状态机 PENDING→RESERVED。
	Reserve(ctx context.Context, orderID int64) error
	// Track 跟踪(order.html 时间轴):返回订单 + 环节日志。
	Track(ctx context.Context, orderID int64) (*Order, []StageLog, error)
}
