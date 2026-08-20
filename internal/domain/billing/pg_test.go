package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListBills(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "bill_no", "customer_id", "customer_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "period", "amount", "status"}
	mock.ExpectQuery(`SELECT id, bill_no, customer_id, customer_name, legal_entity_id, legal_entity_name, region_id, region_name, period, amount, status FROM bills`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "BILL-202608-201", int64(1), "王先生", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", 158.0, "UNPAID"))

	s := NewPGStore(mock)
	got, err := s.ListBills(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if len(got) != 1 || got[0].BillNo != "BILL-202608-201" || got[0].Amount != 158.0 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateBill(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO bills`).
		WithArgs("BILL-202608-202", int64(2), "赵女士", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", 129.0, "UNPAID").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreateBill(context.Background(), Bill{
		BillNo: "BILL-202608-202", CustomerID: 2, CustomerName: "赵女士", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Period: "2026-08", Amount: 129.0, Status: "UNPAID",
	})
	if err != nil {
		t.Fatalf("CreateBill: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetBill(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		cols := []string{"id", "bill_no", "customer_id", "customer_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "period", "amount", "status"}
		mock.ExpectQuery(`SELECT id, bill_no, customer_id`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), "BILL-202608-201", int64(1), "王先生", int64(1), "主品牌·企业", int64(11), "马尼拉市", "2026-08", 158.0, "PAID"))

		s := NewPGStore(mock)
		b, err := s.GetBill(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetBill: %v", err)
		}
		if b.Status != "PAID" {
			t.Fatalf("b=%+v", b)
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

		mock.ExpectQuery(`SELECT id, bill_no`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetBill(context.Background(), 99)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_ListPayments(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "pay_no", "bill_id", "amount", "method", "status"}
	mock.ExpectQuery(`SELECT id, pay_no, COALESCE\(bill_id,0\), amount, method, status FROM payments`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "PAY-20260820-001", int64(1), 158.0, "wechat", "SUCCESS"))

	s := NewPGStore(mock)
	got, err := s.ListPayments(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListPayments: %v", err)
	}
	if len(got) != 1 || got[0].PayNo != "PAY-20260820-001" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreatePayment(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-20260820-002", int64(1), int64(0), 158.0, "alipay", "SUCCESS").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreatePayment(context.Background(), Payment{
		PayNo: "PAY-20260820-002", BillID: 1, Amount: 158.0, Method: "alipay", Status: "SUCCESS",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_GenerateBills 契约:按账期为在网客户批量出账,金额=产品基础月费,幂等。
func TestPGStore_GenerateBills(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO bills`).
		WithArgs("2026-08").
		WillReturnResult(pgxmock.NewResult("INSERT", 3))

	s := NewPGStore(mock)
	n, err := s.GenerateBills(context.Background(), "2026-08")
	if err != nil {
		t.Fatalf("GenerateBills: %v", err)
	}
	if n != 3 {
		t.Fatalf("generated=%d, want 3", n)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
