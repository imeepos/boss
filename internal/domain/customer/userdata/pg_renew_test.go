package userdata

// 续费 SQL 测试:到期月延长表达式参数、未命中 ErrPlanNotFound。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_RenewPlan(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`UPDATE user_plans SET contract_end`).
		WithArgs(int64(9), int64(7), 15).
		WillReturnRows(mock.NewRows([]string{"contract_end"}).AddRow("2027-08"))

	s := NewPGStore(mock)
	end, err := s.RenewPlan(context.Background(), 9, 7, 15)
	if err != nil {
		t.Fatalf("RenewPlan: %v", err)
	}
	if end != "2027-08" {
		t.Fatalf("end=%s", end)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStore_RenewPlanNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`UPDATE user_plans SET contract_end`).
		WithArgs(int64(9), int64(7), 6).
		WillReturnRows(mock.NewRows([]string{"contract_end"}))

	s := NewPGStore(mock)
	if _, err := s.RenewPlan(context.Background(), 9, 7, 6); !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("err=%v want ErrPlanNotFound", err)
	}
}

func TestPGStore_GetPlanForRenewal(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT p.product_id, po.monthly_fee`).
		WithArgs(int64(9), int64(7)).
		WillReturnRows(mock.NewRows([]string{"product_id", "monthly_fee"}).AddRow(int64(5), 199.0))

	s := NewPGStore(mock)
	pid, fee, err := s.GetPlanForRenewal(context.Background(), 9, 7)
	if err != nil || pid != 5 || fee != 199.0 {
		t.Fatalf("GetPlanForRenewal: %v %d %f", err, pid, fee)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
