package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func int64Ptr(v int64) *int64 { return &v }

func TestPGStoreGenerateAgingSnapshots_Idempotent(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`INSERT INTO ar_aging_snapshots`).
		WithArgs("2026-08-24").
		WillReturnResult(pgxmock.NewResult("INSERT", 3))

	s := NewPGStore(pool)
	got, err := s.GenerateAgingSnapshots(context.Background(), time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Fatalf("generated=%d, want 3", got)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreListAgingSnapshots(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	day := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	pool.ExpectQuery(`SELECT id, customer_id, snapshot_date, current_amount, days_1_30, days_31_60, days_61_90, days_90_plus, total_amount FROM ar_aging_snapshots`).
		WithArgs("2026-08-24", int64(0)).
		WillReturnRows(pool.NewRows([]string{"id", "customer_id", "snapshot_date", "current_amount", "days_1_30", "days_31_60", "days_61_90", "days_90_plus", "total_amount"}).
			AddRow(int64(5), int64(9), day, 0.0, 100.0, 50.0, 0.0, 0.0, 150.0).
			AddRow(int64(6), int64(10), day, 0.0, 0.0, 0.0, 200.0, 500.0, 700.0))

	s := NewPGStore(pool)
	items, err := s.ListAgingSnapshots(context.Background(), day, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].CustomerID != 9 || items[0].Days1To30 != 100.0 || items[0].Total != 150.0 {
		t.Fatalf("items=%+v", items)
	}
	if items[1].CustomerID != 10 || items[1].Days61To90 != 200.0 || items[1].Days90Plus != 500.0 {
		t.Fatalf("items[1]=%+v", items[1])
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreCreatePaymentPromise(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	promised := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	due := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	pool.ExpectQuery(`INSERT INTO ar_payment_promises`).
		WithArgs(int64(3), 500.0, promised, due, "OPEN").
		WillReturnRows(pool.NewRows([]string{"id"}).AddRow(int64(7)))

	s := NewPGStore(pool)
	id, err := s.CreatePaymentPromise(context.Background(), PaymentPromise{
		CustomerID: 3, Amount: 500.0, PromisedAt: promised, DueAt: due, Status: "OPEN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != 7 {
		t.Fatalf("id=%d", id)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreUpdatePaymentPromiseStatus(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`UPDATE ar_payment_promises SET status=`).
		WithArgs(int64(7), "FULFILLED").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(pool)
	if err := s.UpdatePaymentPromiseStatus(context.Background(), 7, "FULFILLED"); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreUpdatePaymentPromiseStatus_NotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`UPDATE ar_payment_promises SET status=`).
		WithArgs(int64(99), "BROKEN").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewPGStore(pool)
	if err := s.UpdatePaymentPromiseStatus(context.Background(), 99, "BROKEN"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreListPaymentPromises(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	due := time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC)
	pool.ExpectQuery(`SELECT id, customer_id, amount, promised_at, due_at, status, fulfilled_at FROM ar_payment_promises`).
		WithArgs(int64(3)).
		WillReturnRows(pool.NewRows([]string{"id", "customer_id", "amount", "promised_at", "due_at", "status", "fulfilled_at"}).
			AddRow(int64(7), int64(3), 500.0, due, due, "OPEN", nil))

	s := NewPGStore(pool)
	items, err := s.ListPaymentPromises(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 7 || items[0].Amount != 500.0 || items[0].Status != "OPEN" {
		t.Fatalf("items=%+v", items)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreCreateWriteoff(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectQuery(`INSERT INTO ar_writeoffs`).
		WithArgs(int64(1), 100.0, "坏账", (*int64)(nil)).
		WillReturnRows(pool.NewRows([]string{"id"}).AddRow(int64(4)))

	s := NewPGStore(pool)
	id, err := s.CreateWriteoff(context.Background(), Writeoff{CustomerID: 1, Amount: 100.0, Reason: "坏账"})
	if err != nil {
		t.Fatal(err)
	}
	if id != 4 {
		t.Fatalf("id=%d", id)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreListWriteoffs(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectQuery(`SELECT id, customer_id, amount, reason, approved_by, approved_at FROM ar_writeoffs`).
		WithArgs(int64(1)).
		WillReturnRows(pool.NewRows([]string{"id", "customer_id", "amount", "reason", "approved_by", "approved_at"}).
			AddRow(int64(4), int64(1), 100.0, "坏账", int64Ptr(99), nil))

	s := NewPGStore(pool)
	items, err := s.ListWriteoffs(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 4 || items[0].Amount != 100.0 || items[0].Reason != "坏账" || items[0].ApprovedBy == nil || *items[0].ApprovedBy != 99 {
		t.Fatalf("items=%+v", items)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
