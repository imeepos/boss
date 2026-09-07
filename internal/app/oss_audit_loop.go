package app

// P5-W3 资源台账稽核每日循环:每日 04:00(对账 03:00 之后错开)跑三类稽核,
// 结果落 report_snapshots(period=oss-audit-daily,当日已跑则跳过);
// 失败必须留 [oss-audit] FAILED 级可 grep 日志,禁止静默吞错。
// RESERVED 阈值走 biz_params(resource.audit.reservedStaleHours,默认 48 小时)。

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/internal/domain/resource"
)

const (
	ossAuditLoopInterval = 30 * time.Minute // 错过 04:00 最迟 04:30 补跑
	ossAuditHour         = 4
	ossAuditParamKey     = "resource.audit.reservedStaleHours"
)

// paramReader biz_params 窄口(对齐 reserve_timeout_loop paramLister 先例)。
type paramReader interface {
	GetParam(ctx context.Context, key string) (string, error)
}

// ossAuditDeps 循环最小依赖,便于单测注入。
type ossAuditDeps struct {
	res resource.InventoryAuditor // 必备:稽核执行体
	rep *report.ReportService     // 必备:快照落盘
	pg  paramReader               // 可空:阈值热读,缺省回退默认
}

// startOSSAuditLoop 启动每日稽核循环,返回 stop(幂等);
// 稽核能力/快照服务缺失时为空操作,便于单测与降级部署。
func startOSSAuditLoop(a *Application) (stop func()) {
	aud, ok := a.Resource.(resource.InventoryAuditor)
	if !ok || a.Report == nil {
		return func() {}
	}
	d := ossAuditDeps{res: aud, rep: a.Report}
	if a.User != nil {
		if p, ok := a.User.(paramReader); ok {
			d.pg = p
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(ossAuditLoopInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runOSSAuditIfDue(ctx, d, time.Now())
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

// runOSSAuditIfDue 到点(>=04:00)且当日未跑则执行一轮。
func runOSSAuditIfDue(ctx context.Context, d ossAuditDeps, now time.Time) {
	if now.Hour() < ossAuditHour {
		return
	}
	if ossAuditDoneToday(ctx, d, now) {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	rep, err := runOSSAuditSnapshot(cctx, d, now)
	if err != nil {
		log.Printf("[oss-audit] DAILY SNAPSHOT FAILED: %v", err)
		return
	}
	if rep.Total > 0 {
		log.Printf("[oss-audit] daily findings total=%d staleHours=%d(明细见 report_snapshots/oss-audit-daily)",
			rep.Total, rep.StaleHours)
	}
}

// runOSSAuditSnapshot 执行一轮稽核并落当日快照。
func runOSSAuditSnapshot(ctx context.Context, d ossAuditDeps, now time.Time) (*resource.AuditReport, error) {
	opts := resource.AuditOptions{ReservedStaleHours: ossAuditStaleHours(ctx, d.pg)}
	rep, err := d.res.AuditInventory(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("audit inventory: %w", err)
	}
	if _, err := d.rep.SaveOSSAudit(ctx, rep, now); err != nil {
		return nil, fmt.Errorf("save snapshot: %w", err)
	}
	return rep, nil
}

// ossAuditStaleHours 读 biz_params 阈值;缺失/非法回退默认;读失败留 FAILED 日志。
func ossAuditStaleHours(ctx context.Context, pg paramReader) int {
	if pg == nil {
		return resource.AuditDefaultStaleHours
	}
	v, err := pg.GetParam(ctx, ossAuditParamKey)
	if err != nil {
		log.Printf("[oss-audit] PARAM READ FAILED key=%s err=%v(fallback default %d)",
			ossAuditParamKey, err, resource.AuditDefaultStaleHours)
		return resource.AuditDefaultStaleHours
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return resource.AuditDefaultStaleHours
	}
	return n
}

// ossAuditDoneToday 当日快照已存在(幂等跳过);读失败视为未跑,下一轮重试。
func ossAuditDoneToday(ctx context.Context, d ossAuditDeps, now time.Time) bool {
	snap, err := d.rep.St.LatestSnapshot(ctx, report.PeriodOSSAuditDaily)
	if err != nil {
		return false
	}
	w := snap.WindowStart.In(now.Location())
	return w.Year() == now.Year() && w.YearDay() == now.YearDay()
}
