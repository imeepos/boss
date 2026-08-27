// Webhook 投递循环(M2):周期调用 DeliverDue,清空到期 outbox;失败仅记日志下轮重试。
// outbox 行持久于 PG,重启不丢;DeliverDue 经 ListDue 原子领取(SKIP LOCKED+租约),
// 多实例并发运行同一行不会被重复投递。
package app

import (
	"context"
	"log"
	"time"
)

type webhookDeliverer interface {
	DeliverDue(ctx context.Context) (int, error)
}

const webhookDeliveryInterval = 15 * time.Second

// startWebhookDeliveryLoop 启动投递循环,返回停止函数。
func startWebhookDeliveryLoop(d webhookDeliverer) func() {
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		tk := time.NewTicker(webhookDeliveryInterval)
		defer tk.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				n, err := d.DeliverDue(ctx)
				if err != nil {
					log.Printf("[openplat-webhook] deliver error: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("[openplat-webhook] processed %d deliveries", n)
				}
			}
		}
	}()
	return stop
}
