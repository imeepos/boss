package partner

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_AccrueCommissionIdempotent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`INSERT INTO partner_commission_ledger`).
		WithArgs(int64(11), int64(9), 1000.0, 0.1).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(55)))
	id, err := NewPGStore(mock).AccrueCommission(context.Background(), 11, 9, 1000, 0.1)
	if err != nil || id != 55 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStore_SettleCommissionRequiresTenant(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectExec(`UPDATE partner_commission_ledger`).
		WithArgs(int64(55), int64(7)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if err := NewPGStore(mock).SettleCommissionLedger(context.Background(), 55, 7); err == nil {
		t.Fatal("expected missing tenant settlement error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
