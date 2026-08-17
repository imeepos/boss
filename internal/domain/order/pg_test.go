package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// stubExists 桩 CustomerLookup。
type stubExists struct{ ok bool }

func (s stubExists) Exists(context.Context, int64) (bool, error) { return s.ok, nil }

var ts = time.Date(2025, 8, 17, 10, 0, 0, 0, time.UTC)

func TestPGStore_Submit(t *testing.T) {
	t.Run("成功", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT 'ORD-'`).
			WillReturnRows(mock.NewRows([]string{"order_no"}).AddRow("ORD-20250817-000001"))
		mock.ExpectQuery(`INSERT INTO orders`).
			WithArgs(pgxmock.AnyArg(), int64(1), int64(10), int64(100), int8(1), "PENDING", int64(5), int64(1), "root.luzon").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(7), int8(1), "DOING").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock, stubExists{ok: true})
		o, err := s.Submit(context.Background(), SubmitReq{
			CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5, LegalEntityID: 1, RegionPath: "root.luzon",
		})
		if err != nil {
			t.Fatalf("Submit: %v", err)
		}
		if o.ID != 7 || o.Status != "PENDING" || o.Stage != 1 || o.OrderNo == "" {
			t.Fatalf("o=%+v", o)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("客户不存在", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		s := NewPGStore(mock, stubExists{ok: false})
		_, err = s.Submit(context.Background(), SubmitReq{CustomerID: 99, ChannelID: 5})
		if err == nil {
			t.Fatal("want error for missing customer")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("缺渠道", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		s := NewPGStore(mock, stubExists{ok: true})
		if _, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1}); err == nil {
			t.Fatal("want error for missing channel")
		}
	})
}

func TestPGStore_Reserve(t *testing.T) {
	t.Run("成功", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, status FROM orders`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"stage", "status"}).AddRow(int8(2), "PENDING"))
		mock.ExpectExec(`UPDATE orders SET stage`).
			WithArgs(int64(1), int8(3), "RESERVED").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`INSERT INTO order_stages`).
			WithArgs(int64(1), int8(3), "DONE").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock, stubExists{})
		if err := s.Reserve(context.Background(), 1); err != nil {
			t.Fatalf("Reserve: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("非法流转", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT stage, status FROM orders`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"stage", "status"}).AddRow(int8(2), "RESERVED"))

		s := NewPGStore(mock, stubExists{})
		err = s.Reserve(context.Background(), 1)
		if !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v, want ErrIllegalTransition", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_Track(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	orderCols := []string{"id", "order_no", "customer_id", "offer_id", "address_id", "stage", "status", "channel_id", "legal_entity_id", "region_path", "created_at"}
	mock.ExpectQuery(`SELECT id, order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path, created_at FROM orders`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(orderCols).
			AddRow(int64(1), "ORD-20250817-001", int64(1), int64(10), int64(100), int8(3), "RESERVED", int64(5), int64(1), "root.luzon", ts))
	mock.ExpectQuery(`SELECT id, order_id, stage, result, retries, finished_at FROM order_stages`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "order_id", "stage", "result", "retries", "finished_at"}).
			AddRow(int64(1), int64(1), int8(1), "DONE", int16(0), nil).
			AddRow(int64(2), int64(1), int8(3), "DOING", int16(0), nil))

	s := NewPGStore(mock, stubExists{})
	o, stages, err := s.Track(context.Background(), 1)
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if o.Status != "RESERVED" || len(stages) != 2 || stages[0].Stage != 1 {
		t.Fatalf("o=%+v stages=%+v", o, stages)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_TrackNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, order_no, customer_id`).
		WithArgs(int64(99)).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock, stubExists{})
	_, _, err = s.Track(context.Background(), 99)
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("err=%v, want ErrOrderNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
