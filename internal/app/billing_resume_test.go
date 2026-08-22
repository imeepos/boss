package app

import (
	"context"
	"errors"
	"testing"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/billing"
)

// resumeAaa 桩:仅覆盖复机路径用到的两个方法,其余走嵌入接口(误调即 panic 暴露)。
type resumeAaa struct {
	aaa.AaaService
	lo         *aaa.LoAccount
	resumed    int64
	resumeErr  error
}

func (f *resumeAaa) GetLoAccountByCustomer(context.Context, int64) (*aaa.LoAccount, error) {
	return f.lo, nil
}
func (f *resumeAaa) ResumeLoAccount(_ context.Context, id int64) error {
	f.resumed = id
	return f.resumeErr
}

// resumeArrears 桩:记录追加的停复机流水。
type resumeArrears struct {
	billing.ArrearsService
	tasks []billing.StopResumeTask
}

func (f *resumeArrears) AppendStopResumeTask(_ context.Context, t billing.StopResumeTask) (int64, error) {
	f.tasks = append(f.tasks, t)
	return int64(len(f.tasks)), nil
}

func TestResumeAfterPayment(t *testing.T) {
	t.Run("停机客户缴费即复机并留痕", func(t *testing.T) {
		aaaFake := &resumeAaa{lo: &aaa.LoAccount{ID: 7, Status: "SUSPENDED"}}
		arrearsFake := &resumeArrears{}
		a := &Application{Aaa: aaaFake, Arrears: arrearsFake}

		a.ResumeAfterPayment(context.Background(), 100)

		if aaaFake.resumed != 7 {
			t.Fatalf("resumed=%d, want 7", aaaFake.resumed)
		}
		if len(arrearsFake.tasks) != 1 || arrearsFake.tasks[0].Action != "RESUME" ||
			arrearsFake.tasks[0].Status != "DONE" || arrearsFake.tasks[0].LoAccountID != 7 {
			t.Fatalf("tasks=%+v", arrearsFake.tasks)
		}
	})
	t.Run("在网客户缴费不复机", func(t *testing.T) {
		aaaFake := &resumeAaa{lo: &aaa.LoAccount{ID: 7, Status: "ACTIVE"}}
		arrearsFake := &resumeArrears{}
		a := &Application{Aaa: aaaFake, Arrears: arrearsFake}

		a.ResumeAfterPayment(context.Background(), 100)

		if aaaFake.resumed != 0 || len(arrearsFake.tasks) != 0 {
			t.Fatalf("resumed=%d tasks=%d, want 0/0", aaaFake.resumed, len(arrearsFake.tasks))
		}
	})
	t.Run("复机失败不冒泡留 FAILED 可重试", func(t *testing.T) {
		aaaFake := &resumeAaa{lo: &aaa.LoAccount{ID: 7, Status: "SUSPENDED"}, resumeErr: errors.New("net down")}
		arrearsFake := &resumeArrears{}
		a := &Application{Aaa: aaaFake, Arrears: arrearsFake}

		a.ResumeAfterPayment(context.Background(), 100)

		if len(arrearsFake.tasks) != 1 || arrearsFake.tasks[0].Status != "FAILED" {
			t.Fatalf("tasks=%+v, want FAILED", arrearsFake.tasks)
		}
	})
	t.Run("无 LO 账号静默跳过", func(t *testing.T) {
		aaaFake := &resumeAaa{} // GetLoAccountByCustomer 返回 nil,nil
		arrearsFake := &resumeArrears{}
		a := &Application{Aaa: aaaFake, Arrears: arrearsFake}

		a.ResumeAfterPayment(context.Background(), 100)

		if len(arrearsFake.tasks) != 0 {
			t.Fatalf("tasks=%d, want 0", len(arrearsFake.tasks))
		}
	})
	t.Run("客户 0 直接返回", func(t *testing.T) {
		a := &Application{}
		a.ResumeAfterPayment(context.Background(), 0) // 不应 panic
	})
}
