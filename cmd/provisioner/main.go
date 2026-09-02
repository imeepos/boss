// provisioner 服务入口(W7):轮询下发任务 → 设备协议执行 → 状态机迁移 + 留痕。
// driver 三态(BOSS_PROVISION_DRIVER):log(默认日志桩)/telnet(OLT 行协议)/tl1(U2000 TL1)。
package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/provision/tl1"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

// logExecutor 降级桩:OLT 地址未配置时仅记录(协议适配前的安全默认)。
type logExecutor struct{}

func (logExecutor) Exec(ctx context.Context, t provision.Task) (provision.ExecTrace, error) {
	log.Printf("provisioner: exec task %d (template=%d, noop)", t.ID, t.TemplateID)
	return provision.ExecTrace{Commands: []string{
		fmt.Sprintf("noop driver=log template=%d task=%s event=%s", t.TemplateID, t.TaskNo, t.StageEvent),
	}, Response: "log driver noop(未配置设备驱动,未实际下发)"}, nil
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

	exec := selectExecutor(cfg, pool)
	d := provision.NewDaemon(provision.NewPGStore(pool), exec, cfg.Provisioner.Interval)
	d.OnDone = provisionNotify(notify.NewPGStore(pool))
	log.Println("provisioner: started")
	d.Run(ctx)
	log.Println("provisioner: stopped")
}

// selectExecutor 按 driver 选择设备执行器;OnDone 通知逻辑三态共用(调用方统一挂接)。
func selectExecutor(cfg *config.Config, pool *pgxpool.Pool) provision.Executor {
	switch cfg.Provisioner.Driver {
	case "tl1":
		log.Printf("provisioner: tl1 executor: endpoint resolved per task")
		return tl1.NewExecutor(app.NewTL1ParamResolver(pool))
	case "telnet":
		if cfg.Provisioner.OLTAddr == "" || cfg.Provisioner.OLTAddr == "0" {
			log.Printf("provisioner: telnet driver but OLT_ADDR unset, fallback log")
			return logExecutor{}
		}
		log.Printf("provisioner: telnet executor -> %s", cfg.Provisioner.OLTAddr)
		return &provision.TelnetExecutor{
			Addr: cfg.Provisioner.OLTAddr, User: cfg.Provisioner.OLTUser, Pass: cfg.Provisioner.OLTPass,
			Timeout: 5 * time.Second,
		}
	default: // log(默认桩,零影响升级)
		return logExecutor{}
	}
}
