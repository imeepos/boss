package provision

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// W7 写侧(TDD 先行):任务执行(装维零手工)+ 失败重试留痕。

func TestPGStore_ExecuteTask(t *testing.T) {
	t.Run("PENDING→DONE 直调完成", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(1), "DONE", []string{"DOING", "PENDING"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT t\.id, COALESCE\(t\.code,.*FROM provision_tasks tk`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"id", "code"}).AddRow(int64(16), "TPL-FTTH"))
		mock.ExpectQuery(`INSERT INTO provision_logs`).
			WithArgs(int64(1), int64(0), "", int64(16), "TPL-FTTH", "SUCCESS", int16(0), "provision apply template=16", "OK", "telnet").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))

		s := NewPGStore(mock)
		if err := s.ExecuteTask(context.Background(), 1, ExecTrace{Commands: []string{"provision apply template=16"}, Response: "OK", Driver: DriverTelnet}); err != nil {
			t.Fatalf("ExecuteTask: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("非 DOING/PENDING 拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(2), "DONE", []string{"DOING", "PENDING"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.ExecuteTask(context.Background(), 2, ExecTrace{}); !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestPGStore_FailTask(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`UPDATE provision_tasks SET status`).
		WithArgs(int64(3), "FAILED", []string{"DOING", "PENDING"}).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectQuery(`SELECT t\.id, COALESCE\(t\.code,.*FROM provision_tasks tk`).
		WithArgs(int64(3)).
		WillReturnRows(mock.NewRows([]string{"id", "code"}).AddRow(int64(16), "TPL-FTTH"))
	mock.ExpectQuery(`INSERT INTO provision_logs`).
		WithArgs(int64(3), int64(0), "", int64(16), "TPL-FTTH", "FAILED: olt connect timeout", int16(0), "", "", "tl1").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))

	s := NewPGStore(mock)
	if err := s.FailTask(context.Background(), 3, "olt connect timeout", ExecTrace{Driver: DriverTL1}); err != nil {
		t.Fatalf("FailTask: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ClaimTask(t *testing.T) {
	t.Run("领取 PENDING 任务成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		cols := []string{"id", "task_no", "order_id", "stage_event", "lo_account_id", "template_id", "status"}
		mock.ExpectQuery(`UPDATE provision_tasks SET status`).
			WillReturnRows(mock.NewRows(cols).AddRow(int64(1), "TASK-1", int64(9), "preConfigOLT", int64(88), int64(1), "DOING"))
		s := NewPGStore(mock)
		task, err := s.ClaimTask(context.Background())
		if err != nil || task == nil {
			t.Fatalf("claim failed: err=%v task=%v", err, task)
		}
		if task.ID != 1 || task.Status != "DOING" {
			t.Fatalf("task=%+v", task)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("无待办返回 nil", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`UPDATE provision_tasks SET status`).
			WillReturnRows(pgxmock.NewRows([]string{"id", "task_no", "order_id", "stage_event", "lo_account_id", "template_id", "status"}))
		s := NewPGStore(mock)
		task, err := s.ClaimTask(context.Background())
		if err != nil || task != nil {
			t.Fatalf("expected nil task, got task=%v err=%v", task, err)
		}
	})
}

func TestPGStore_RetryTask(t *testing.T) {
	t.Run("FAILED→PENDING + 重试计数", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(3), "PENDING", []string{"FAILED"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`SELECT t\.id, COALESCE\(t\.code,.*FROM provision_tasks tk`).
			WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"id", "code"}).AddRow(int64(16), "TPL-FTTH"))
		mock.ExpectQuery(`INSERT INTO provision_logs`).
			WithArgs(int64(3), int64(0), "", int64(16), "TPL-FTTH", "RETRY", int16(2), "", "", "").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(12)))

		s := NewPGStore(mock)
		if err := s.RetryTask(context.Background(), 3, 1); err != nil {
			t.Fatalf("RetryTask: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("非 FAILED 拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(4), "PENDING", []string{"FAILED"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.RetryTask(context.Background(), 4, 0); !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v", err)
		}
	})
}
