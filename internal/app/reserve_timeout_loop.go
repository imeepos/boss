package app

// Q2 订单超时释放:预占超时后台循环。周期扫描 RESERVED 超时订单,
// 阈值走 biz_params(order.reserve.timeoutMinutes,默认 30,roadmap §2),
// 释放后逐单调 task 通知留痕(补偿任务中心可回放视角的第一块)。

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/user"
)

// expiredReserveReleaser OrderService 的可选超时释放能力(PGStore 实现;
// 按 OrphanPatroller 模式窄口断言,不污染 OrderService 接口)。
type expiredReserveReleaser interface {
	ReleaseExpiredReserves(ctx context.Context, cutoff time.Time) ([]int64, error)
}

// paramLister User 服务可选能力:读 biz_params(热更阈值)。
type paramLister interface {
	ListParams(ctx context.Context) ([]user.Param, error)
}

// reserveTimeoutDeps 循环依赖的最小集合,便于单测注入。
type reserveTimeoutDeps struct {
	rel expiredReserveReleaser // 必备:超时释放
	l   paramLister            // 可空:阈值热更,缺省回退默认
	n   notify.Service         // 可空:释放留痕
}

const (
	reserveTimeoutInterval = time.Minute
	reserveTimeoutParamKey = "order.reserve.timeoutMinutes"
	reserveTimeoutDefault  = 30
	reserveTimeoutMin      = 1
	reserveTimeoutMax      = 24 * 60
)

// startReserveTimeoutLoop 启动预占超时释放循环,返回 stop(幂等)。
// Order 不具备该能力时为空操作,便于单测与降级部署。
func startReserveTimeoutLoop(a *Application) (stop func()) {
	rel, ok := a.Order.(expiredReserveReleaser)
	if !ok {
		return func() {}
	}
	d := reserveTimeoutDeps{rel: rel}
	if a.User != nil {
		if l, ok := a.User.(paramLister); ok {
			d.l = l
		}
	}
	d.n = a.Notify
	return runReserveTimeoutLoop(d)
}

func runReserveTimeoutLoop(d reserveTimeoutDeps) (stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runReserveTimeoutOnce(ctx, d)
		t := time.NewTicker(reserveTimeoutInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runReserveTimeoutOnce(ctx, d)
			}
		}
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
}

func runReserveTimeoutOnce(ctx context.Context, d reserveTimeoutDeps) {
	cctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cutoff := time.Now().Add(-time.Duration(reserveTimeoutMinutes(cctx, d.l)) * time.Minute)
	ids, err := d.rel.ReleaseExpiredReserves(cctx, cutoff)
	if err != nil {
		log.Printf("reserve timeout loop: %v", err)
	}
	for _, id := range ids {
		emitReserveReleaseNotice(cctx, d, id)
	}
}

// reserveTimeoutMinutes 读 biz_params 阈值;库不可读/值非法回退默认,钳到 [1,1440]。
func reserveTimeoutMinutes(ctx context.Context, l paramLister) int {
	if l == nil {
		return reserveTimeoutDefault
	}
	list, err := l.ListParams(ctx)
	if err != nil {
		return reserveTimeoutDefault
	}
	for _, p := range list {
		if p.Key != reserveTimeoutParamKey {
			continue
		}
		v, err := strconv.Atoi(p.Value)
		if err != nil || v < reserveTimeoutMin {
			return reserveTimeoutDefault
		}
		if v > reserveTimeoutMax {
			return reserveTimeoutMax
		}
		return v
	}
	return reserveTimeoutDefault
}

// emitReserveReleaseNotice 超时释放留痕:task 通知(refType/refID 幂等),
// 尽力而为,失败只记日志。
func emitReserveReleaseNotice(ctx context.Context, d reserveTimeoutDeps, orderID int64) {
	if d.n == nil {
		return
	}
	in := notify.Input{
		Category: notify.CategoryTask,
		Level:    notify.LevelWarn,
		Title:    "预占超时自动释放",
		Content:  "订单 " + strconv.FormatInt(orderID, 10) + " 预占超时,端口已回收,订单回 PENDING 待重新核查",
		Link:     "/boss/order",
		RefType:  "order_reserve_timeout",
		RefID:    strconv.FormatInt(orderID, 10),
	}
	if err := d.n.Emit(ctx, in); err != nil {
		log.Printf("reserve timeout loop: emit %d: %v", orderID, err)
	}
}
