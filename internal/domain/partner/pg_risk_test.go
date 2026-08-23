package partner

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CheckOrderRiskDailyCap(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT COALESCE\(legal_entity_id,0\) FROM accounts`).WithArgs(int64(7)).WillReturnRows(mock.NewRows([]string{"entity"}).AddRow(int64(9)))
	mock.ExpectQuery(`SELECT`).WithArgs(int64(9), int64(11)).WillReturnRows(mock.NewRows([]string{"daily", "recent"}).AddRow(int64(DefaultDailyOrderLimit), int64(0)))
	got, err := NewPGStore(mock).CheckOrderRisk(context.Background(), 7, 11)
	if err != nil {
		t.Fatal(err)
	}
	if got.Allowed || got.Reason != "DAILY_ORDER_LIMIT" {
		t.Fatalf("decision=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStore_CheckOrderRiskCooldown(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT COALESCE\(legal_entity_id,0\) FROM accounts`).WithArgs(int64(7)).WillReturnRows(mock.NewRows([]string{"entity"}).AddRow(int64(9)))
	mock.ExpectQuery(`SELECT`).WithArgs(int64(9), int64(11)).WillReturnRows(mock.NewRows([]string{"daily", "recent"}).AddRow(int64(1), int64(1)))
	got, err := NewPGStore(mock).CheckOrderRisk(context.Background(), 7, 11)
	if err != nil {
		t.Fatal(err)
	}
	if got.Allowed || got.Reason != "CUSTOMER_COOLDOWN" {
		t.Fatalf("decision=%+v", got)
	}
}
