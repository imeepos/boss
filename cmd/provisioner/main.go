// provisioner 服务入口(W7):轮询下发任务 → 设备协议执行 → 状态机迁移 + 留痕。
// 债务偿还:默认走真实 Telnet 执行器(BOSS_PROVISION_OLT_ADDR 配置),未配置时降级日志桩。
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

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
	log.Println("provisioner: started")
	d.Run(ctx)
	log.Println("provisioner: stopped")
}
