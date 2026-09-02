package provision

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProvSvc struct {
	ProvisionService
	tasks   []Task
	claimAt int // 按序领取,超量返回 nil
	failed  []string
	done    []int64
}

func (f *fakeProvSvc) ClaimTask(context.Context) (*Task, error) {
	for i := f.claimAt; i < len(f.tasks); i++ {
		if f.tasks[i].Status == "PENDING" {
			f.claimAt = i + 1
			t := f.tasks[i]
			t.Status = "DOING"
			return &t, nil
		}
	}
	return nil, nil
}
func (f *fakeProvSvc) FailTask(_ context.Context, id int64, reason string, _ ExecTrace) error {
	f.failed = append(f.failed, reason)
	return nil
}
func (f *fakeProvSvc) ExecuteTask(_ context.Context, id int64, _ ExecTrace) error {
	f.done = append(f.done, id)
	return nil
}

type fakeExec struct{ err error }

func (e fakeExec) Exec(context.Context, Task) (ExecTrace, error) { return ExecTrace{}, e.err }

func TestDaemonTick(t *testing.T) {
	t.Run("执行成功 → ExecuteTask", func(t *testing.T) {
		svc := &fakeProvSvc{tasks: []Task{{ID: 1, Status: "PENDING"}, {ID: 2, Status: "DONE"}}}
		d := NewDaemon(svc, fakeExec{}, time.Second)
		d.tick(context.Background())
		if len(svc.done) != 1 || svc.done[0] != 1 {
			t.Fatalf("done=%v", svc.done)
		}
	})

	t.Run("执行失败 → FailTask 留痕", func(t *testing.T) {
		svc := &fakeProvSvc{tasks: []Task{{ID: 1, Status: "PENDING"}}}
		d := NewDaemon(svc, fakeExec{err: errors.New("olt timeout")}, time.Second)
		d.tick(context.Background())
		if len(svc.failed) != 1 || svc.failed[0] != "olt timeout" {
			t.Fatalf("failed=%v", svc.failed)
		}
	})
}
