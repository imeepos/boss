package order

import (
	"context"
	"errors"
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

// TestPGStore_UpdateMapRepairsReservedPort 回归:订单已 DONE 但端口仍 RESERVED 时，重试环节12必须补偿为 USED。
func TestPGStore_UpdateMapRepairsReservedPort(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(12), "DONE", "ORD-7"))
	mock.ExpectQuery(`SELECT stage, status FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status"}).AddRow(int8(12), "DONE"))
	mock.ExpectExec(`UPDATE ports SET status = 'USED'`).
		WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock, stubExists{ok: true})
	if err := s.UpdateMap(context.Background(), 7); err != nil {
		t.Fatalf("UpdateMap retry: %v", err)
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

// TestPGStore_RollbackStage 回归(ISSUE.md worker rollback 仅审计不落库):
// 删最新环节日志、stage 前移、status 级联逆向、工单随动 DOING。
func TestPGStore_RollbackStage(t *testing.T) {
	t.Run("段11回退:DONE→INSTALLING + 工单随动", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, status FROM orders`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"stage", "status"}).AddRow(int8(11), "DONE"))
		mock.ExpectExec(`DELETE FROM order_stages`).
			WithArgs(int64(7), int8(11)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectExec(`UPDATE orders SET stage`).
			WithArgs(int64(7), int8(10), "INSTALLING").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`UPDATE dispatch_tickets`).
			WithArgs(int64(7), "DOING").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock, stubExists{ok: true})
		if err := s.RollbackStage(context.Background(), 7); err != nil {
			t.Fatalf("RollbackStage: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("stage<2 拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT stage, status FROM orders`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"stage", "status"}).AddRow(int8(1), "PENDING"))
		s := NewPGStore(mock, stubExists{ok: true})
		if err := s.RollbackStage(context.Background(), 7); err != ErrIllegalTransition {
			t.Fatalf("err=%v, want ErrIllegalTransition", err)
		}
	})
}

// fakePrepaidCollector 预付费收款桩:记录入参,可注入失败与赠送返回。
type fakePrepaidCollector struct {
	customerID int64
	amount     float64
	offerID    int64
	months     int
	gift       int
	err        error
}

func (f *fakePrepaidCollector) Collect(_ context.Context, customerID int64, amount float64,
	offerID int64, months int) (int, error) {
	f.customerID, f.amount, f.offerID, f.months = customerID, amount, offerID, months
	return f.gift, f.err
}

// expectChargeAdvance 环节4 推进(advance)的 mock 序列:select → update → 环节日志。
func expectChargeAdvance(mock pgxmock.PgxPoolIface) {
	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(3), "RESERVED", "ORD-7"))
	mock.ExpectExec(`UPDATE orders SET stage`).
		WithArgs(int64(7), int8(4), "RESERVED").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO order_stages`).
		WithArgs(int64(7), int8(4), "DONE").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

// TestPGStore_ChargeContractPrepaid 契约:预付费订单环节4 先当场收款再推进(REQ-CL-001);
// 预缴 12 月按 12x月费 收,赠送命中回填 orders.gift_months。
func TestPGStore_ChargeContractPrepaid(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectChargeSelect(mock, "PREPAID", 9, 5, 12)
	mock.ExpectQuery(`SELECT COALESCE\(ro.monthly_fee, po.monthly_fee\)`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"amount"}).AddRow(199.0))
	mock.ExpectExec(`UPDATE orders SET gift_months`).
		WithArgs(int64(7), 3).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	expectChargeAdvance(mock)

	fake := &fakePrepaidCollector{gift: 3}
	s := NewPGStore(mock, stubExists{ok: true}, fake)
	if err := s.ChargeContract(context.Background(), 7); err != nil {
		t.Fatalf("ChargeContract: %v", err)
	}
	if fake.customerID != 9 || fake.amount != 199.0*12 || fake.offerID != 5 || fake.months != 12 {
		t.Fatalf("collect=%+v", fake)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_ChargeContractPrepaidMonthly 契约:buy_months=0(按月缴)收 1 个月月费,不回填赠送。
func TestPGStore_ChargeContractPrepaidMonthly(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectChargeSelect(mock, "PREPAID", 9, 5, 0)
	mock.ExpectQuery(`SELECT COALESCE\(ro.monthly_fee, po.monthly_fee\)`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"amount"}).AddRow(199.0))
	expectChargeAdvance(mock)

	fake := &fakePrepaidCollector{}
	s := NewPGStore(mock, stubExists{ok: true}, fake)
	if err := s.ChargeContract(context.Background(), 7); err != nil {
		t.Fatalf("ChargeContract: %v", err)
	}
	if fake.amount != 199.0 || fake.months != 1 {
		t.Fatalf("collect=%+v", fake)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// expectChargeSelect 环节4 前置查询桩:付费模式/客户/产品/预缴月数。
func expectChargeSelect(mock pgxmock.PgxPoolIface, mode string, customerID, offerID, buyMonths int64) {
	mock.ExpectQuery(`SELECT billing_mode, customer_id, offer_id, buy_months FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"billing_mode", "customer_id", "offer_id", "buy_months"}).
			AddRow(mode, customerID, offerID, buyMonths))
}

// TestPGStore_ChargeContractPostpaidSkipsCollect 契约:后付费不触发当场收款。
func TestPGStore_ChargeContractPostpaidSkipsCollect(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectChargeSelect(mock, "POSTPAID", 9, 5, 0)
	expectChargeAdvance(mock)

	fake := &fakePrepaidCollector{}
	s := NewPGStore(mock, stubExists{ok: true}, fake)
	if err := s.ChargeContract(context.Background(), 7); err != nil {
		t.Fatalf("ChargeContract: %v", err)
	}
	if fake.customerID != 0 {
		t.Fatalf("postpaid should not collect, got %+v", fake)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_ChargeContractPrepaidCollectFail 契约:收款失败环节4 不推进(未收费不派单)。
func TestPGStore_ChargeContractPrepaidCollectFail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectChargeSelect(mock, "PREPAID", 9, 5, 0)
	mock.ExpectQuery(`SELECT COALESCE\(ro.monthly_fee, po.monthly_fee\)`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"amount"}).AddRow(199.0))

	fake := &fakePrepaidCollector{err: errors.New("pay channel down")}
	s := NewPGStore(mock, stubExists{ok: true}, fake)
	if err := s.ChargeContract(context.Background(), 7); err == nil {
		t.Fatal("want error when collect fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
