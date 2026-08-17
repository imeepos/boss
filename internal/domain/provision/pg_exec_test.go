package provision

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// W7 写侧(TDD 先行):任务执行(装维零手工)+ 失败重试留痕。

func TestPGStore_ExecuteTask(t *testing.T) {
	t.Run("PENDING→DOING→DONE 双段迁移+日志", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(1), "DOING", []string{"PENDING"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(1), "DONE", []string{"DOING"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`INSERT INTO provision_logs`).
			WithArgs(int64(1), int64(0), "", int64(0), "", "SUCCESS", int16(0)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))

		s := NewPGStore(mock)
		if err := s.ExecuteTask(context.Background(), 1); err != nil {
			t.Fatalf("ExecuteTask: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("非 PENDING 拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(2), "DOING", []string{"PENDING"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.ExecuteTask(context.Background(), 2); !errors.Is(err, ErrIllegalTransition) {
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
	mock.ExpectQuery(`INSERT INTO provision_logs`).
		WithArgs(int64(3), int64(0), "", int64(0), "", "FAILED: olt connect timeout", int16(0)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))

	s := NewPGStore(mock)
	if err := s.FailTask(context.Background(), 3, "olt connect timeout"); err != nil {
		t.Fatalf("FailTask: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_RetryTask(t *testing.T) {
	t.Run("FAILED→PENDING + 重试计数", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE provision_tasks SET status`).
			WithArgs(int64(3), "PENDING", []string{"FAILED"}).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectQuery(`INSERT INTO provision_logs`).
			WithArgs(int64(3), int64(0), "", int64(0), "", "RETRY", int16(2)).
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
