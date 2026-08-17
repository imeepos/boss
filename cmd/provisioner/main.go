// provisioner 服务入口(W7):轮询下发任务 → 设备协议执行 → 状态机迁移 + 留痕。
// Executor 当前为日志桩;OLT 协议(Telnet/SNMP-set)接入时替换装配即可。
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

// logExecutor 桩执行器:记录下发动作(协议适配前的安全默认)。
type logExecutor struct{}

func (logExecutor) Exec(ctx context.Context, t provision.Task) error {
	log.Printf("provisioner: exec task %d (template=%d)", t.ID, t.TemplateID)
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

	d := provision.NewDaemon(provision.NewPGStore(pool), logExecutor{}, 5*time.Second)
	log.Println("provisioner: started")
	d.Run(ctx)
	log.Println("provisioner: stopped")
}
