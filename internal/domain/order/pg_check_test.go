package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_CheckResource 契约:环节2 资源核查推进 stage=2;无资源停在 stage=2 等待。
func TestPGStore_CheckResource(t *testing.T) {
	t.Run("有资源推进", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, address_id FROM orders`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"stage", "address_id"}).AddRow(int8(1), int64(100)))
		mock.ExpectExec(`UPDATE orders SET stage`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(7), int8(2), "DONE").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock, stubExists{}, &stubChecker{available: true})
		if err := s.CheckResource(context.Background(), 7); err != nil {
			t.Fatalf("CheckResource: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("无资源停在stage2", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, address_id FROM orders`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"stage", "address_id"}).AddRow(int8(1), int64(100)))
		mock.ExpectExec(`UPDATE orders SET stage`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(7), int8(2), "PENDING").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock, stubExists{}, &stubChecker{available: false})
		if err := s.CheckResource(context.Background(), 7); err != nil {
			t.Fatalf("CheckResource: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("幂等", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, address_id FROM orders`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"stage", "address_id"}).AddRow(int8(2), int64(100)))

		s := NewPGStore(mock, stubExists{}, &stubChecker{available: true})
		if err := s.CheckResource(context.Background(), 7); err != nil {
			t.Fatalf("CheckResource: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("订单不存在", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, address_id FROM orders`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock, stubExists{}, &stubChecker{available: true})
		if err := s.CheckResource(context.Background(), 99); !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("err=%v, want ErrOrderNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}
