package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 调拨审批/驳回 + 手动释放预占:阶段4 台账写侧(TDD 先行)。

func TestPGStore_ReviewTransfer(t *testing.T) {
	t.Run("approve PENDING→DOING", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE transfers SET status = 'DOING'`).
			WithArgs("TRF-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.ApproveTransfer(context.Background(), "TRF-1"); err != nil {
			t.Fatalf("ApproveTransfer: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("reject PENDING→DONE(终态)", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE transfers SET status = 'DONE'`).
			WithArgs("TRF-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.RejectTransfer(context.Background(), "TRF-1"); err != nil {
			t.Fatalf("RejectTransfer: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("单号不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE transfers`).
			WithArgs("TRF-X").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.ApproveTransfer(context.Background(), "TRF-X"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

func TestPGStore_ReleaseReserve(t *testing.T) {
	t.Run("释放 HELD 记录并回收端口", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FOR UPDATE`).
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"port_id", "order_id", "status"}).AddRow(int64(3), int64(42), "HELD"))
		mock.ExpectExec(`UPDATE ports SET status = 'IDLE', order_id = NULL`).
			WithArgs(int64(42)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`UPDATE reserve_records SET status = 'RELEASED'`).
			WithArgs(int64(9)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.ReleaseReserve(context.Background(), 9); err != nil {
			t.Fatalf("ReleaseReserve: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("记录不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FOR UPDATE`).
			WithArgs(int64(99)).
			WillReturnRows(mock.NewRows([]string{"port_id", "order_id", "status"}))
		s := NewPGStore(mock)
		if err := s.ReleaseReserve(context.Background(), 99); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
