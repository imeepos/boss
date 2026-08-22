package app

// Q2 每日数据对账循环:每日 03:00(roadmap §2 数据核对周期)跑五域对账,
// 当日已有快照则跳过(幂等);异常发 URGENT 待办(P1 责任队列入口,
// refType=daily_recon, refID=当日,每日幂等)。

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/report"
)

const (
	reconLoopInterval = 30 * time.Minute // 对齐检查粒度:错过 03:00 最迟 03:30 补跑
	reconHour         = 3
)

// reconLoopDeps 循环最小依赖,便于单测注入。
type reconLoopDeps struct {
	rep *report.ReportService // 必备:对账执行体
	n   notify.Service        // 可空:异常待办
	now func() time.Time
}

// startDailyReconLoop 启动每日对账循环,返回 stop(幂等)。
func startDailyReconLoop(a *Application) (stop func()) {
	if a.Report == nil {
		return func() {}
	}
	d := reconLoopDeps{rep: a.Report, n: a.Notify, now: time.Now}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(reconLoopInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runDailyReconIfDue(ctx, d)
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

// runDailyReconIfDue 到点(>=03:00)且当日未跑则执行一轮。
func runDailyReconIfDue(ctx context.Context, d reconLoopDeps) {
	now := d.now()
	if now.Hour() < reconHour {
		return
	}
	if reconDoneToday(ctx, d, now) {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	_, payload, err := d.rep.DailyRecon(cctx, now)
	if err != nil {
		if err != report.ErrReconUnsupported {
			log.Printf("daily recon: %v", err)
		}
		return
	}
	if !payload.AllOK {
		emitReconNotice(cctx, d, payload, now)
	}
}

// reconDoneToday 当日快照已存在(幂等跳过);读失败视为未跑,下一轮重试。
func reconDoneToday(ctx context.Context, d reconLoopDeps, now time.Time) bool {
	snap, err := d.rep.St.LatestSnapshot(ctx, report.PeriodReconDaily)
	if err != nil {
		return false
	}
	w := snap.WindowStart.In(now.Location())
	return w.Year() == now.Year() && w.YearDay() == now.YearDay()
}

// emitReconNotice 异常待办:URGENT todo,列失败域;每日幂等。
func emitReconNotice(ctx context.Context, d reconLoopDeps, p *report.ReconPayload, now time.Time) {
	if d.n == nil {
		return
	}
	var bad []string
	for _, c := range p.Checks {
		if !c.OK() {
			bad = append(bad, c.Domain+"."+c.Name+"="+strconv.FormatInt(c.Count, 10))
		}
	}
	in := notify.Input{
		Category: notify.CategoryTodo,
		Level:    notify.LevelUrgent,
		Title:    "每日数据对账发现异常",
		Content:  "异常项: " + strings.Join(bad, ", ") + ";请按 P1 时限处理(四码冲突 4 小时内清零)",
		Link:     "/intel/report",
		RefType:  "daily_recon",
		RefID:    now.Format("20060102"),
	}
	if err := d.n.Emit(ctx, in); err != nil {
		log.Printf("daily recon: emit: %v", err)
	}
}
