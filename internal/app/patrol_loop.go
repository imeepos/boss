package app

// 孤儿巡检定时循环(db-design-review D7 定时化):每小时整点跑一次
// Report.PatrolSnapshot 落 report_snapshots;失败仅记日志不重试不中断。
// 随 Application 生命周期启停(Close 时 stop)。

import (
	"context"
	"log"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/order"
)

// patrolInterval 巡检周期;整点对齐只是观感,错过由下一轮补上。
const patrolInterval = time.Hour

// startPatrolLoop 启动后台巡检循环,返回 stop(幂等)。
// 首轮按错峰延迟执行一次(启动即有当日基线),此后按周期。
func startPatrolLoop(a *Application) (stop func()) {
	if a.Report == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		staggeredFirstRun(ctx, startupDelays["patrol"], func(c context.Context) { runPatrolOnce(c, a) })
		t := time.NewTicker(patrolInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runPatrolOnce(ctx, a)
			}
		}
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second): // 不阻塞退出
		}
	}
}

func runPatrolOnce(ctx context.Context, a *Application) {
	cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if _, err := a.Report.PatrolSnapshot(cctx, time.Now()); err != nil {
		log.Printf("patrol loop: %v", err)
	}
	patrolOverdueComplaints(cctx, a)
	patrolEscalateOverdueTodos(cctx, a)
}

// patrolEscalateOverdueTodos P1 待办超时升级:超时未办就地升 URGENT(000115);
// 每小时一轮,幂等只动 level。日志留痕,不额外刷通知。
func patrolEscalateOverdueTodos(ctx context.Context, a *Application) {
	esc, ok := a.Notify.(notify.OverdueEscalator)
	if !ok || a.Notify == nil {
		return
	}
	n, err := esc.EscalateOverdue(ctx)
	if err != nil {
		log.Printf("patrol escalate todos: %v", err)
		return
	}
	if n > 0 {
		log.Printf("patrol escalate todos: %d overdue todo(s) escalated to URGENT", n)
	}
}

// patrolOverdueComplaints 报障工单超时巡检:SLA 已过且未办结 → URGENT 待办提醒
// (ref=complaint/<ticketNo> 幂等,每小时一轮不重复刷)。办结时 admin 端 Resolve。
func patrolOverdueComplaints(ctx context.Context, a *Application) {
	if a.Notify == nil || a.WorkOrder == nil {
		return
	}
	complaints, err := a.WorkOrder.ListComplaints(ctx)
	if err != nil {
		log.Printf("patrol complaints: list: %v", err)
		return
	}
	for _, c := range complaints {
		if !overdueComplaint(c) {
			continue
		}
		err := a.Notify.Emit(ctx, notify.Input{
			Category: notify.CategoryTodo, Level: notify.LevelUrgent,
			Title: "报障工单超时:" + c.TicketNo, RefType: "complaint",
			RefID: c.TicketNo, Link: "/boss/complaint",
		})
		if err != nil {
			log.Printf("patrol complaints: emit %s: %v", c.TicketNo, err)
		}
	}
}

// overdueComplaint 未办结且 SLA 截止时间已过(sla_deadline 格式 YYYY-MM-DD HH24:MI)。
func overdueComplaint(c order.Complaint) bool {
	if c.Status == "CLOSED" || c.SlaDeadline == "" {
		return false
	}
	dl, err := time.ParseInLocation("2006-01-02 15:04", c.SlaDeadline, time.Local)
	return err == nil && time.Now().After(dl)
}
