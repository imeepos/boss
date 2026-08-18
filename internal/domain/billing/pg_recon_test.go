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
