package app

import (
	"context"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/billing"
)

// dunningFake 桩:欠费清单+欠费快照+停复机流水,LO 状态可变。
type dunningFake struct {
	billing.DunningService
	billing.ArrearsService // 嵌入接口兜底未覆盖方法(误调即 panic 暴露)
	marked                 int
	overdue                []billing.OverdueCustomer
	arrears                map[int64]billing.Arrears
	lo                     map[int64]*aaa.LoAccount
	suspended              []int64
	tasks                  []billing.StopResumeTask
}

func (f *dunningFake) MarkOverdueBills(context.Context, int) (int, error) {
	return f.marked, nil
}
func (f *dunningFake) ListOverdueCustomers(context.Context) ([]billing.OverdueCustomer, error) {
	return f.overdue, nil
}
func (f *dunningFake) UpsertArrears(_ context.Context, a billing.Arrears) (int64, error) {
	f.arrears[a.CustomerID] = a
	return a.CustomerID, nil
}
func (f *dunningFake) AppendStopResumeTask(_ context.Context, t billing.StopResumeTask) (int64, error) {
	f.tasks = append(f.tasks, t)
	return int64(len(f.tasks)), nil
}

type dunningAaa struct {
	aaa.AaaService
	f *dunningFake
}

func (d *dunningAaa) GetLoAccountByCustomer(_ context.Context, cid int64) (*aaa.LoAccount, error) {
	return d.f.lo[cid], nil
}
func (d *dunningAaa) SuspendLoAccount(_ context.Context, id int64) error {
	d.f.suspended = append(d.f.suspended, id)
	return nil
}

func TestRunDunning(t *testing.T) {
	now := time.Now().UTC()
	t.Run("超停机线自动停机+欠费快照", func(t *testing.T) {
		f := &dunningFake{
			marked:  3,
			overdue: []billing.OverdueCustomer{{CustomerID: 213, Amount: 1998, OldestAt: now.AddDate(0, 0, -40)}},
			arrears: map[int64]billing.Arrears{},
			lo:      map[int64]*aaa.LoAccount{213: {ID: 92, CustomerID: 213, Status: "ACTIVE"}},
		}
		a := &Application{Dunning: f, Arrears: f, Aaa: &dunningAaa{f: f}}

		res, err := a.RunDunning(context.Background(), 15, 30)
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if res.OverdueBills != 3 || len(res.StoppedIDs) != 1 || res.StoppedIDs[0] != 213 {
			t.Fatalf("res=%+v", res)
		}
		ar := f.arrears[213]
		if ar.Status != billing.ArrearsStopped || ar.Amount != 1998 || ar.Days < 39 {
			t.Fatalf("arrears=%+v", ar)
		}
		if len(f.suspended) != 1 || f.suspended[0] != 92 || len(f.tasks) != 1 ||
			f.tasks[0].Action != "STOP" || f.tasks[0].Status != "DONE" {
			t.Fatalf("suspended=%v tasks=%+v", f.suspended, f.tasks)
		}
	})
	t.Run("未到停机线只催收", func(t *testing.T) {
		f := &dunningFake{
			overdue: []billing.OverdueCustomer{{CustomerID: 1, Amount: 500, OldestAt: now.AddDate(0, 0, -20)}},
			arrears: map[int64]billing.Arrears{},
			lo:      map[int64]*aaa.LoAccount{1: {ID: 9, Status: "ACTIVE"}},
		}
		a := &Application{Dunning: f, Arrears: f, Aaa: &dunningAaa{f: f}}

		res, err := a.RunDunning(context.Background(), 15, 30)
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if len(res.StoppedIDs) != 0 || f.arrears[1].Status != billing.ArrearsCollecting {
			t.Fatalf("res=%+v arrears=%+v", res, f.arrears[1])
		}
		if len(f.suspended) != 0 {
			t.Fatalf("suspended=%v, want none", f.suspended)
		}
	})
	t.Run("LO 非 ACTIVE 不重复停机", func(t *testing.T) {
		f := &dunningFake{
			overdue: []billing.OverdueCustomer{{CustomerID: 2, Amount: 100, OldestAt: now.AddDate(0, 0, -40)}},
			arrears: map[int64]billing.Arrears{},
			lo:      map[int64]*aaa.LoAccount{2: {ID: 8, Status: "SUSPENDED"}},
		}
		a := &Application{Dunning: f, Arrears: f, Aaa: &dunningAaa{f: f}}

		res, _ := a.RunDunning(context.Background(), 15, 30)
		if len(res.StoppedIDs) != 0 || len(f.suspended) != 0 || len(f.tasks) != 0 {
			t.Fatalf("res=%+v suspended=%v tasks=%v", res, f.suspended, f.tasks)
		}
	})
	t.Run("Dunning 未装配返回空结果", func(t *testing.T) {
		a := &Application{}
		res, err := a.RunDunning(context.Background(), 15, 30)
		if err != nil || res.OverdueBills != 0 {
			t.Fatalf("res=%+v err=%v", res, err)
		}
	})
}
