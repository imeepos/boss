package app

// 后台循环启动期错峰:多个"启动即首轮"的循环若同时执行,会在进程启动瞬间
// 集中打满 DB/下游。为各循环分配互不相同的固定首轮延迟,只推迟首轮开始时刻,
// 不改变执行频率、判定语义与后续节奏;数值仅按错峰需要排布,无业务含义。

import (
	"context"
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
func staggeredFirstRun(ctx context.Context, offset time.Duration, fn func(context.Context)) bool {
	if offset <= 0 {
		fn(ctx)
		return true
	}
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
