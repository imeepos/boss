package app

// AAA-A2 在线会话后台循环:Disconnect 有限次重试 + 僵尸会话周期清理。
// 判定与迁移逻辑在 aaa.SessionControlService(域层可单测),此处只管节拍与生命周期。

import (
	"context"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

const (
	offlineRetryInterval = 30 * time.Second // PENDING_OFFLINE 重试节拍
	offlineRetryBatch    = 100              // 每轮重试上限
	zombieScanInterval   = 10 * time.Minute // 僵尸扫描节拍
	zombieScanBatch      = 500              // 每轮清理上限
)

// startAAAOfflineRetryLoop PENDING_OFFLINE 会话重试循环(下发失败/重试耗尽转终态)。
func startAAAOfflineRetryLoop(svc *aaa.SessionControlService) (stop func()) {
	return startSessionTicker("aaa_offline_retry", svc, offlineRetryInterval,
		func(c context.Context) { svc.RetryPendingOffline(c, offlineRetryBatch) })
}

// startZombieReapLoop 僵尸会话清理循环:超时未更新的 ONLINE 会话关闭并补录 Stop 话单。
func startZombieReapLoop(svc *aaa.SessionControlService, zombieAfter time.Duration) (stop func()) {
	if zombieAfter <= 0 {
		zombieAfter = 2 * time.Hour
	}
	return startSessionTicker("aaa_zombie_reap", svc, zombieScanInterval,
		func(c context.Context) {
			_, _ = svc.ReapZombieSessions(c, time.Now().Add(-zombieAfter), zombieScanBatch)
		})
}

// startSessionTicker 通用节拍器:首轮错峰 → 固定间隔执行;ctx 取消即停(幂等)。
func startSessionTicker(name string, svc *aaa.SessionControlService, interval time.Duration, once func(context.Context)) (stop func()) {
	if svc == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		staggeredFirstRun(ctx, name, startupDelays[name], func(c context.Context) { once(c) })
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				once(ctx)
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
