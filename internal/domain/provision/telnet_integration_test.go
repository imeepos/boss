package provision

// TelnetExecutor + Daemon 真实 PG 集成测试(债务偿还验收):下发守护进程轮询 PENDING →
// 真实 TCP 执行器(loopback fake OLT) → 状态迁移 DONE + 留痕,失败路径 FAILED 留痕。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/domain/provision/ -run TestDaemonTelnet -v -count=1

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// filteredSvc 只让 tick 处理指定 taskID(隔离其它可能并发的 PENDING 任务)。
type filteredSvc struct {
	ProvisionService
	taskID int64
}

func (f *filteredSvc) ClaimTask(ctx context.Context) (*Task, error) {
	all, err := f.ProvisionService.ListTasks(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range all {
		if t.ID == f.taskID && t.Status == "PENDING" {
			t.Status = "DOING"
			return &t, nil
		}
	}
	return nil, nil
}

func TestDaemonTelnet_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewPGStore(pool)

	newTask := func(t *testing.T, execOK bool) (int64, string) {
		t.Helper()
		suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e12)
		tplID, err := store.CreateTemplate(ctx, Template{LegalEntityID: 1, Code: "TPL-TN-" + suffix, Name: "tnet"})
		if err != nil {
			t.Fatal(err)
		}
		taskNo := "TASK-TN-" + suffix
		id, err := store.CreateTask(ctx, Task{TaskNo: taskNo, LoAccountID: 999999, TemplateID: tplID, Status: "PENDING"})
		if err != nil {
			t.Fatal(err)
		}
		f := startFakeOLT(t, map[bool]string{true: "OK", false: "ERR config invalid"}[execOK])
		exec := &TelnetExecutor{Addr: f.addr(), User: "admin", Pass: "secret"}
		NewDaemon(&filteredSvc{ProvisionService: store, taskID: id}, exec, 0).tick(ctx)
		return id, taskNo
	}

	t.Run("成功下发生成 SUCCESS 留痕", func(t *testing.T) {
		id, taskNo := newTask(t, true)
		tk, err := store.GetTaskByNo(ctx, taskNo)
		if err != nil {
			t.Fatal(err)
		}
		if tk.Status != "DONE" {
			t.Fatalf("status=%s, want DONE", tk.Status)
		}
		logs, err := store.ListLogs(ctx, id)
		if err != nil || len(logs) == 0 || logs[0].Result != "SUCCESS" {
			t.Fatalf("logs=%+v err=%v", logs, err)
		}
	})

	t.Run("OLT 失败生成 FAILED 留痕", func(t *testing.T) {
		id, taskNo := newTask(t, false)
		tk, _ := store.GetTaskByNo(ctx, taskNo)
		if tk.Status != "FAILED" {
			t.Fatalf("status=%s, want FAILED", tk.Status)
		}
		logs, _ := store.ListLogs(ctx, id)
		found := false
		for _, l := range logs {
			if strings.HasPrefix(l.Result, "FAILED:") {
				found = true
			}
		}
		if !found {
			t.Fatalf("no FAILED log in %+v", logs)
		}
	})
}
