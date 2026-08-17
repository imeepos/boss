package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_List 契约:联表返回客户/产品/地址名,按 keyword/status 过滤。
func TestPGStore_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT o.order_no, COALESCE\(c.name`).
		WithArgs("", "PENDING", 1<<30, 0).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer", "product", "address", "stage", "status", "created_at"}).
			AddRow("ORD-20250817-001", "王先生", "100M宽带", "Manila", int8(3), "PENDING", ts))

	s := NewPGStore(mock, stubExists{})
	got, err := s.List(context.Background(), OrderQuery{Status: "PENDING"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0].OrderNo != "ORD-20250817-001" || got[0].Customer != "王先生" || got[0].Stage != 3 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_GetByNo 契约:按订单号查;未命中返回 ErrOrderNotFound。
func TestPGStore_GetByNo(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, order_no, customer_id, offer_id, address_id, stage, status, channel_id, legal_entity_id, region_path, created_at FROM orders WHERE order_no`).
			WithArgs("ORD-20250817-001").
			WillReturnRows(mock.NewRows([]string{"id", "order_no", "customer_id", "offer_id", "address_id", "stage", "status", "channel_id", "legal_entity_id", "region_path", "created_at"}).
				AddRow(int64(7), "ORD-20250817-001", int64(1), int64(10), int64(100), int8(3), "PENDING", int64(5), int64(1), "root.luzon", ts))

		s := NewPGStore(mock, stubExists{})
		o, err := s.GetByNo(context.Background(), "ORD-20250817-001")
		if err != nil {
			t.Fatalf("GetByNo: %v", err)
		}
		if o.ID != 7 || o.OrderNo != "ORD-20250817-001" || o.Stage != 3 {
			t.Fatalf("o=%+v", o)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, order_no`).
			WithArgs("ORD-NONE").
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock, stubExists{})
		_, err = s.GetByNo(context.Background(), "ORD-NONE")
		if !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("err=%v, want ErrOrderNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
