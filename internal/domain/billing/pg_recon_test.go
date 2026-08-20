package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_ListReconciliations 契约:批次列表含 diff 派生值。
func TestPGStore_ListReconciliations(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	settled := time.Date(2025, 8, 16, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT id, batch_no, channel, channel_amount, system_amount, status, created_at, settled_at
		FROM reconciliation_batches`).
		WillReturnRows(mock.NewRows([]string{
			"id", "batch_no", "channel", "channel_amount", "system_amount", "status", "created_at", "settled_at",
		}).
			AddRow(int64(2), "PC-20250816-04", "支付宝", 862005.00, 861905.00, "DIFF_PENDING", settled, nil).
			AddRow(int64(1), "PC-20250816-05", "微信", 1284320.00, 1284320.00, "SETTLED", settled, &settled))

	s := NewPGStore(mock)
	list, err := s.ListReconciliations(context.Background())
	if err != nil {
		t.Fatalf("ListReconciliations: %v", err)
	}
	if len(list) != 2 || list[0].Diff != 100.00 || list[1].Diff != 0 || list[1].SettledAt == nil {
		t.Fatalf("list=%+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_SettleReconciliation 契约:差异挂起批次平账;已平账再平账返回非法迁移。
func TestPGStore_SettleReconciliation(t *testing.T) {
	t.Run("平账成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE reconciliation_batches SET status='SETTLED'`).
			WithArgs("PC-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).SettleReconciliation(context.Background(), "PC-1"); err != nil {
			t.Fatalf("SettleReconciliation: %v", err)
		}
	})
	t.Run("已平账拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE reconciliation_batches`).WithArgs("PC-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectQuery(`SELECT id, batch_no, channel`).
			WithArgs("PC-1").
			WillReturnRows(mock.NewRows([]string{
				"id", "batch_no", "channel", "channel_amount", "system_amount", "status", "created_at", "settled_at",
			}).AddRow(int64(1), "PC-1", "微信", 1.0, 1.0, "SETTLED", time.Now(), nil))
		err := NewPGStore(mock).SettleReconciliation(context.Background(), "PC-1")
		if !errors.Is(err, ErrIllegalReconTransition) {
			t.Fatalf("err=%v", err)
		}
	})
}

// ptrArg 匹配任意指向该值的 *int64(pgxmock 不按解引用值比对指针)。
type ptrArg int64

func (p ptrArg) Match(v interface{}) bool {
	pv, ok := v.(*int64)
	return ok && pv != nil && *pv == int64(p)
}

// TestCompareStatement 契约:四种 diff_kind 判定——双边一致/渠道有系统无/系统有渠道无/金额不一致。
func TestCompareStatement(t *testing.T) {
	rows := []ChannelStatementRow{
		{ChannelRef: "PAY-1", Amount: 100},
		{ChannelRef: "PAY-2", Amount: 100},
		{ChannelRef: "CH-ONLY", Amount: 50},
		{ChannelRef: "PAY-3", Amount: 30},
	}
	pays := []PaymentRef{
		{ID: 11, PayNo: "PAY-1", Amount: 100},
		{ID: 12, PayNo: "PAY-2", Amount: 80},
		{ID: 13, PayNo: "PAY-4", Amount: 60},
	}
	items := CompareStatement(rows, pays)
	byRef := map[string]ReconItem{}
	for _, it := range items {
		byRef[it.ChannelRef] = it
	}
	if k := byRef["PAY-1"]; k.DiffKind != DiffMatch || k.PaymentID == nil || *k.PaymentID != 11 {
		t.Fatalf("PAY-1=%+v", k)
	}
	if k := byRef["PAY-2"]; k.DiffKind != DiffAmountMismatch || k.PaymentID == nil || *k.PaymentID != 12 {
		t.Fatalf("PAY-2=%+v", k)
	}
	if k := byRef["CH-ONLY"]; k.DiffKind != DiffMissingSystem || k.PaymentID != nil {
		t.Fatalf("CH-ONLY=%+v", k)
	}
	if k := byRef["PAY-4"]; k.DiffKind != DiffMissingChannel || k.PaymentID == nil || *k.PaymentID != 13 {
		t.Fatalf("PAY-4=%+v", k)
	}
	if len(items) != 5 {
		t.Fatalf("items=%+v", items)
	}
}

// TestPGStore_RecordChannelStatement 契约:录入渠道流水→比对落 items→重算批次总额与状态。
func TestPGStore_RecordChannelStatement(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	created := time.Date(2025, 8, 16, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`FROM reconciliation_batches WHERE id=\$1`).WithArgs(int64(4)).
		WillReturnRows(mock.NewRows([]string{
			"id", "batch_no", "channel", "channel_amount", "system_amount", "status", "created_at", "settled_at",
		}).AddRow(int64(4), "PC-20250816-04", "支付宝", 0, 0, "DIFF_PENDING", created, nil))
	mock.ExpectQuery(`FROM payments`).WithArgs(created).
		WillReturnRows(mock.NewRows([]string{"id", "pay_no", "amount"}).
			AddRow(int64(11), "PAY-1", 100.00))
	mock.ExpectExec(`DELETE FROM reconciliation_items`).WithArgs(int64(4)).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectExec(`INSERT INTO reconciliation_items`).
		WithArgs(int64(4), ptrArg(11), "PAY-1", 100.00, "MATCH", "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`UPDATE reconciliation_batches SET channel_amount`).
		WithArgs(int64(4), 100.00, 100.00, "SETTLED").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if err := NewPGStore(mock).RecordChannelStatement(context.Background(), 4,
		[]ChannelStatementRow{{ChannelRef: "PAY-1", Amount: 100}}); err != nil {
		t.Fatalf("RecordChannelStatement: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_RecordChannelStatement_SettledRejected 契约:已平账批次拒绝再录渠道流水。
func TestPGStore_RecordChannelStatement_SettledRejected(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	created := time.Date(2025, 8, 16, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`FROM reconciliation_batches WHERE id=\$1`).WithArgs(int64(4)).
		WillReturnRows(mock.NewRows([]string{
			"id", "batch_no", "channel", "channel_amount", "system_amount", "status", "created_at", "settled_at",
		}).AddRow(int64(4), "PC-1", "微信", 1, 1, "SETTLED", created, &created))
	err := NewPGStore(mock).RecordChannelStatement(context.Background(), 4, nil)
	if !errors.Is(err, ErrIllegalReconTransition) {
		t.Fatalf("err=%v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListReconciliationItems 契约:按批次查明细,字段含软引用 payment_id。
func TestPGStore_ListReconciliationItems(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	var payID int64 = 11
	mock.ExpectQuery(`FROM reconciliation_items WHERE batch_id=\$1`).WithArgs(int64(4)).
		WillReturnRows(mock.NewRows([]string{"id", "batch_id", "payment_id", "channel_ref", "amount", "diff_kind", "note"}).
			AddRow(int64(1), int64(4), payID, "PAY-1", 100.00, "MATCH", "").
			AddRow(int64(2), int64(4), nil, "CH-9", 50.00, "MISSING_SYSTEM", ""))

	items, err := NewPGStore(mock).ListReconciliationItems(context.Background(), 4)
	if err != nil {
		t.Fatalf("ListReconciliationItems: %v", err)
	}
	if len(items) != 2 || items[0].DiffKind != DiffMatch || items[1].PaymentID != nil {
		t.Fatalf("items=%+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_GetStopResumeTask 契约:按 id 查停复机任务;未命中返回 ErrNotFound。
func TestPGStore_GetStopResumeTask(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT id, customer_id, lo_account_id, action, status FROM stop_resume_tasks`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"id", "customer_id", "lo_account_id", "action", "status"}).
			AddRow(int64(7), int64(9), int64(88), "STOP", "FAILED"))

	s := NewPGStore(mock)
	task, err := s.GetStopResumeTask(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetStopResumeTask: %v", err)
	}
	if task.Action != "STOP" || task.Status != "FAILED" {
		t.Fatalf("task=%+v", task)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_UpdateStopResumeStatus 契约:重试结果回写;未命中返回 ErrNotFound。
func TestPGStore_UpdateStopResumeStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec(`UPDATE stop_resume_tasks SET status=\$2 WHERE id=\$1`).
		WithArgs(int64(7), "DONE").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := NewPGStore(mock).UpdateStopResumeStatus(context.Background(), 7, "DONE"); err != nil {
		t.Fatalf("UpdateStopResumeStatus: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
