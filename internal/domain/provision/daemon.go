package provision

// W7 provisioner 守护进程:轮询 PENDING 任务 → 设备协议执行 → 状态机迁移 + 留痕。
// 装维零手工:下发/重试全自动;执行器失败可重试(RetryTask 计数)。

import (
	"context"
	"log"
	"time"
)

// Executor 设备协议执行口(OLT/ONU 配置下发;实现可为 Telnet/SNMP-set/gRPC 到网元)。
type Executor interface {
	// Exec 执行一次下发;返回 error 视为失败(写 FAILED 留痕)。
	Exec(ctx context.Context, t Task) error
}

// Daemon 下发守护进程。
type Daemon struct {
	svc  ProvisionService
	exec Executor
	// claim 前置:任务从 PENDING 占位为 DOING 由 ExecuteTask 首段完成,这里先执行后落账。
	interval time.Duration
	// OnDone 任务终态回调(成功/失败各一次);nil=不通知。wiring 层接 notify 域。
	OnDone func(ctx context.Context, t Task, execErr error)
}

// NewDaemon 构造守护进程;interval 为轮询周期。
func NewDaemon(svc ProvisionService, exec Executor, interval time.Duration) *Daemon {
	return &Daemon{svc: svc, exec: exec, interval: interval}
}

// Run 阻塞轮询直至 ctx 取消。
func (d *Daemon) Run(ctx context.Context) {
	t := time.NewTicker(d.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			d.tick(ctx)
		}
	}
}

// tick 一轮:处理全部 PENDING 任务。
func (d *Daemon) tick(ctx context.Context) {
	tasks, err := d.svc.ListTasks(ctx)
	if err != nil {
		log.Printf("provision daemon: list tasks: %v", err)
		return
	}
	for _, t := range tasks {
		if t.Status != "PENDING" {
			continue
		}
		if err := d.exec.Exec(ctx, t); err != nil {
			if failErr := d.svc.FailTask(ctx, t.ID, err.Error()); failErr != nil {
				log.Printf("provision daemon: fail task %d: %v", t.ID, failErr)
			}
			d.notifyDone(ctx, t, err)
			continue
		}
		if err := d.svc.ExecuteTask(ctx, t.ID); err != nil {
			log.Printf("provision daemon: execute task %d: %v", t.ID, err)
			continue
		}
		d.notifyDone(ctx, t, nil)
	}
}

// notifyDone 触发终态回调;回调自身异常只记日志。
func (d *Daemon) notifyDone(ctx context.Context, t Task, execErr error) {
	if d.OnDone == nil {
		return
	}
	d.OnDone(ctx, t, execErr)
}
