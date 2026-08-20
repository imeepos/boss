package userdata

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListUserFaqs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM user_faqs`).
		WillReturnRows(mock.NewRows([]string{"faqId", "category", "question", "answer", "active"}).
			AddRow("FAQ-01", "billing", "如何开发票?", "账单页自助开具", true))

	s := NewPGStore(mock)
	got, err := s.ListUserFaqs(context.Background())
	if err != nil {
		t.Fatalf("ListUserFaqs: %v", err)
	}
	if len(got) != 1 || got[0]["faqId"] != "FAQ-01" || got[0]["active"] != true {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListUsers_Keyword(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`FROM customers c`).
		WithArgs("王").
		WillReturnRows(mock.NewRows([]string{"customerId", "name", "phone", "balance", "activeOrders"}).
			AddRow(int64(1), "王先生", "13800000000", int64(5000), int64(0)))

	s := NewPGStore(mock)
	got, err := s.ListUsers(context.Background(), "王")
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(got) != 1 || got[0]["name"] != "王先生" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ToggleAddon(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE addons`).
		WithArgs("ADD-01").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.ToggleAddon(context.Background(), "ADD-01"); err != nil {
		t.Fatalf("ToggleAddon: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ToggleAddon_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE addons`).
		WithArgs("ADD-404").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewPGStore(mock)
	if err := s.ToggleAddon(context.Background(), "ADD-404"); err != ErrNotFound {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestPGStore_AdjustUserBalance(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 裁定 D1: 余额权威态在 portal_wallets。
	mock.ExpectExec(`INSERT INTO portal_wallets`).
		WithArgs(int64(9), int64(-300)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(mock)
	if err := s.AdjustUserBalance(context.Background(), 9, -300); err != nil {
		t.Fatalf("AdjustUserBalance: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateUserPlan(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO user_plans`).
		WithArgs(int64(8), int64(3), "融合套餐", "ACTIVE", nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.CreateUserPlan(context.Background(), UserPlan{
		CustomerID: 8, ProductID: 3, PlanName: "融合套餐",
	})
	if err != nil {
		t.Fatalf("CreateUserPlan: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
