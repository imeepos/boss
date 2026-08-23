package app

// 积分过期清算循环(2028 Q2 完整 LOY):周期调用 loy.ExpireDue,
// 到期获得流水按客户汇总扣减并落 EXPIRED 负流水;失败仅记日志下轮重试。

import (
	"context"
	"log"
	"time"
)

type pointsExpirer interface {
	ExpireDue(ctx context.Context) (int, error)
}

const pointsExpireInterval = time.Hour

// startPointsExpireLoop 启动积分过期循环,返回停止函数。
func startPointsExpireLoop(p pointsExpirer) func() {
	ctx, stop := context.WithCancel(context.Background())
	go func() {
		tk := time.NewTicker(pointsExpireInterval)
		defer tk.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				n, err := p.ExpireDue(ctx)
				if err != nil {
					log.Printf("[loy-expire] sweep error: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("[loy-expire] expired points for %d customers", n)
				}
			}
		}
	}()
	return stop
}
