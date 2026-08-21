// provisioner 服务入口(W7):轮询下发任务 → 设备协议执行 → 状态机迁移 + 留痕。
// 债务偿还:默认走真实 Telnet 执行器(BOSS_PROVISION_OLT_ADDR 配置),未配置时降级日志桩。
package main

import (
	"context"
	"log"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// logExecutor 降级桩:OLT 地址未配置时仅记录(协议适配前的安全默认)。
type logExecutor struct{}

func (logExecutor) Exec(ctx context.Context, t provision.Task) error {
	log.Printf("provisioner: exec task %d (template=%d, noop)", t.ID, t.TemplateID)
	return nil
}

// provisionNotify 任务终态 → 后台提醒(成功 INFO/失败 WARN;ref=provision/<taskID> 幂等)。
func provisionNotify(ns notify.Service) func(context.Context, provision.Task, error) {
	return func(ctx context.Context, t provision.Task, execErr error) {
		in := notify.Input{
			Category: notify.CategoryTask, Level: notify.LevelInfo,
			Title: "下发任务完成:" + t.TaskNo, RefType: "provision",
			RefID: strconv.FormatInt(t.ID, 10), Link: "/provision/provlog",
		}
		if execErr != nil {
			in.Level = notify.LevelWarn
			in.Title = "下发任务失败:" + t.TaskNo
		}
		if err := ns.Emit(ctx, in); err != nil {
			log.Printf("provisioner: notify emit: %v", err)
		}
	}
}

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("provisioner: open database: %v", err)
	}
	defer pool.Close()

	var exec provision.Executor = logExecutor{}
	if cfg.Provisioner.OLTAddr != "0" && cfg.Provisioner.OLTAddr != "" {
		exec = &provision.TelnetExecutor{
			Addr: cfg.Provisioner.OLTAddr, User: cfg.Provisioner.OLTUser, Pass: cfg.Provisioner.OLTPass,
			Timeout: 5 * time.Second,
		}
		log.Printf("provisioner: telnet executor -> %s", cfg.Provisioner.OLTAddr)
	}

	d := provision.NewDaemon(provision.NewPGStore(pool), exec, cfg.Provisioner.Interval)
	d.OnDone = provisionNotify(notify.NewPGStore(pool))
	log.Println("provisioner: started")
	d.Run(ctx)
	log.Println("provisioner: stopped")
}
