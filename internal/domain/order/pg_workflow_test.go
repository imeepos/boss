package order

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_CancelReleasesPorts 回归:取消订单必须回收本订单预占端口(RESERVED→IDLE),
// 否则端口死占泄漏(修复前 Cancel 只做 status 迁移,ReleasePortByOrder 全仓库无调用方)。
func TestPGStore_CancelReleasesPorts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT status FROM orders`).
		WithArgs(int64(7)).WillReturnRows(mock.NewRows([]string{"status"}).AddRow("RESERVED"))
	mock.ExpectExec(`UPDATE orders SET status`).
		WithArgs(int64(7), "CANCELLED").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// transitionStatus 会同步终态到派单工单。
	mock.ExpectExec(`UPDATE dispatch_tickets`).
		WithArgs(int64(7), "CANCELED").WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	// 关键断言:预占端口被回收。
	mock.ExpectExec(`UPDATE ports SET status = 'IDLE', order_id = NULL`).
		WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock, stubExists{ok: true})
	if err := s.Cancel(context.Background(), 7); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestFullWorkflow 契约:12 环节按序走完,状态沿 PENDING→RESERVED→INSTALLING→DONE 流转。
func TestFullWorkflow(t *testing.T) {
	s, _ := newSvc(1)
	ctx := context.Background()
	o, err := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	id := o.ID

	steps := []struct {
		name string
		fn   func() error
	}{
		{"环节2 资源核查", func() error { return s.CheckResource(ctx, id) }},
		{"环节3 端口预占", func() error { return s.Reserve(ctx, id) }},
		{"环节4 合同收费", func() error { return s.ChargeContract(ctx, id) }},
		{"环节5 标签预绑定", func() error { return s.ApplyTag(ctx, id) }},
		{"环节6 创建账号", func() error { return s.CreateUserProfile(ctx, id) }},
		{"环节7 预下发", func() error { return s.PreConfigOLT(ctx, id) }},
		{"环节8 派单", func() error { return s.DispatchOrder(ctx, id) }},
		{"环节9 扫码绑定", func() error { return s.ScanBind(ctx, id) }},
		{"环节10 激活", func() error { return s.ActivateUser(ctx, id) }},
		{"环节11 激活回调", func() error { return s.NotifyActivation(ctx, id) }},
		{"环节12 更新GIS", func() error { return s.UpdateMap(ctx, id) }},
	}
	for _, st := range steps {
		if err := st.fn(); err != nil {
			t.Fatalf("%s: %v", st.name, err)
		}
	}

	got, logs, err := s.Track(ctx, id)
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if got.Stage != 12 || got.Status != "DONE" {
		t.Fatalf("stage=%d status=%q, want 12/DONE", got.Stage, got.Status)
	}
	if len(logs) != 12 {
		t.Fatalf("环节日志=%d, want 12(每环节一条)", len(logs))
	}
}

// TestWorkflowSequenceGuard 契约:环节不可跳步/不可重复。
func TestWorkflowSequenceGuard(t *testing.T) {
	s, _ := newSvc(1)
	ctx := context.Background()
	o, _ := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})

	// 跳过环节2/3,直接环节4 → 拒绝(stage=1,目标4需 stage=3)。
	if err := s.ChargeContract(ctx, o.ID); err != ErrIllegalTransition {
		t.Fatalf("跳步 err=%v, want ErrIllegalTransition", err)
	}
	// 未收费直接派单(环节8)→ 拒绝。
	if err := s.DispatchOrder(ctx, o.ID); err != ErrIllegalTransition {
		t.Fatalf("未收费派单 err=%v, want ErrIllegalTransition", err)
	}
}

// TestCancelAndRelease 契约:取消/释放只做 status 迁移,不改环节。
func TestCancelAndRelease(t *testing.T) {
	ctx := context.Background()

	t.Run("取消", func(t *testing.T) {
		s, _ := newSvc(1)
		o, _ := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		if err := s.Cancel(ctx, o.ID); err != nil {
			t.Fatalf("Cancel: %v", err)
		}
		got, _, _ := s.Track(ctx, o.ID)
		if got.Status != "CANCELLED" || got.Stage != 1 {
			t.Fatalf("status=%q stage=%d, want CANCELLED/1", got.Status, got.Stage)
		}
		// 已取消不可再取消。
		if err := s.Cancel(ctx, o.ID); err != ErrIllegalTransition {
			t.Fatalf("重复取消 err=%v, want ErrIllegalTransition", err)
		}
	})

	t.Run("释放", func(t *testing.T) {
		s, _ := newSvc(1)
		o, _ := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		s.CheckResource(ctx, o.ID)
		s.Reserve(ctx, o.ID)
		if err := s.Release(ctx, o.ID); err != nil {
			t.Fatalf("Release: %v", err)
		}
		got, _, _ := s.Track(ctx, o.ID)
		if got.Status != "PENDING" || got.Stage != 3 {
			t.Fatalf("status=%q stage=%d, want PENDING/3", got.Status, got.Stage)
		}
	})

	t.Run("已完成不可取消", func(t *testing.T) {
		s, _ := newSvc(1)
		o, _ := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
		for _, fn := range []func() error{
			func() error { return s.CheckResource(ctx, o.ID) },
			func() error { return s.Reserve(ctx, o.ID) },
			func() error { return s.ChargeContract(ctx, o.ID) },
			func() error { return s.ApplyTag(ctx, o.ID) },
			func() error { return s.CreateUserProfile(ctx, o.ID) },
			func() error { return s.PreConfigOLT(ctx, o.ID) },
			func() error { return s.DispatchOrder(ctx, o.ID) },
			func() error { return s.ScanBind(ctx, o.ID) },
			func() error { return s.ActivateUser(ctx, o.ID) },
			func() error { return s.NotifyActivation(ctx, o.ID) },
			func() error { return s.UpdateMap(ctx, o.ID) },
		} {
			if err := fn(); err != nil {
				t.Fatalf("workflow: %v", err)
			}
		}
		if err := s.Cancel(ctx, o.ID); err != ErrIllegalTransition {
			t.Fatalf("DONE 取消 err=%v, want ErrIllegalTransition", err)
		}
	})
}
