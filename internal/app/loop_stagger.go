package app

// 后台循环启动期错峰:多个"启动即首轮"的循环若同时执行,会在进程启动瞬间
// 集中打满 DB/下游。为各循环分配互不相同的固定首轮延迟,只推迟首轮开始时刻,
// 不改变执行频率、判定语义与后续节奏;数值仅按错峰需要排布,无业务含义。

import (
	"context"
	"log"
	"time"
)

// startupDelays 含"启动即首轮"的循环 → 首轮延迟。
// reserve_timeout 不登记,保持启动立即补偿(释放超时预占最时效敏感)。
var startupDelays = map[string]time.Duration{
	"cdr_compensation": 20 * time.Second,
	"etl_executor":     40 * time.Second,
	"patrol":           60 * time.Second,
}

// staggeredFirstRun 等待 offset(或 ctx 取消)后执行 fn;返回是否已执行。
// 记录首轮延迟日志,供实机核对启动期执行分布。
func staggeredFirstRun(ctx context.Context, name string, offset time.Duration, fn func(context.Context)) bool {
	if offset <= 0 {
		log.Printf("[loop-stagger] %s first run immediate", name)
		fn(ctx)
		return true
	}
	log.Printf("[loop-stagger] %s first run deferred by %s", name, offset)
	t := time.NewTimer(offset)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		fn(ctx)
		return true
	}
}
