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

func TestPGStoreApproveWriteoff(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`UPDATE ar_writeoffs SET status='APPROVED', approved_by=`).
		WithArgs(int64(4), int64(99)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(pool)
	if err := s.ApproveWriteoff(context.Background(), 4, 99); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreRejectWriteoff(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`UPDATE ar_writeoffs SET status='REJECTED', approved_note=`).
		WithArgs(int64(4), "not valid").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(pool)
	if err := s.RejectWriteoff(context.Background(), 4, "not valid"); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreEvaluateCreditProfile(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectQuery(`SELECT.*FROM arrears_sum`).
		WithArgs(int64(5)).
		WillReturnRows(pool.NewRows([]string{"customer_id", "credit_level", "risk_score", "stop_threshold", "evaluated_at"}).
			AddRow(int64(5), "WATCH", 1.5, 1000.0, time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)))
	pool.ExpectExec(`INSERT INTO ar_credit_profiles`).
		WithArgs(int64(5), "WATCH", 1.5, 1000.0, time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(pool)
	cp, err := s.EvaluateCreditProfile(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if cp.CreditLevel != "WATCH" {
		t.Fatalf("creditLevel=%s, want WATCH", cp.CreditLevel)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreGenerateCollectionTasks(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`INSERT INTO ar_collection_tasks`).
		WithArgs("2026-08-24").
		WillReturnResult(pgxmock.NewResult("INSERT", 2))

	s := NewPGStore(pool)
	got, err := s.GenerateCollectionTasks(context.Background(), time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("generated=%d", got)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreRecordReplayEvent(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`INSERT INTO ar_replay_events`).
		WithArgs(int64(5), "payment", int64(10), "PAID", `{"amount":100}`).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	s := NewPGStore(pool)
	if err := s.RecordReplayEvent(context.Background(), 5, "payment", 10, "PAID", `{"amount":100}`); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPGStoreReplayEvents(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectQuery(`SELECT id, customer_id, source_type, source_id, event_type, payload::text, created_at FROM ar_replay_events`).
		WithArgs("payment").
		WillReturnRows(pool.NewRows([]string{"id", "customer_id", "source_type", "source_id", "event_type", "payload", "created_at"}).
			AddRow(int64(1), int64(5), "payment", int64(10), "PAID", `{"amount":100}`, time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)))
	pool.ExpectExec(`UPDATE arrears a SET amount`).
		WithArgs(int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(pool)
	got, err := s.ReplayEvents(context.Background(), "payment")
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("replayed=%d", got)
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

	pool.ExpectQuery(`SELECT id, customer_id, amount, reason, COALESCE\(approved_by,0\), approved_at, status, COALESCE\(requested_by,0\), COALESCE\(approved_note,''\) FROM ar_writeoffs`).
		WithArgs(int64(1)).
		WillReturnRows(pool.NewRows([]string{"id", "customer_id", "amount", "reason", "approved_by", "approved_at", "status", "requested_by", "approved_note"}).
			AddRow(int64(4), int64(1), 100.0, "坏账", int64Ptr(99), nil, "APPROVED", int64Ptr(0), ""))

	s := NewPGStore(pool)
	items, err := s.ListWriteoffs(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != 4 || items[0].Amount != 100.0 || items[0].Reason != "坏账" || items[0].ApprovedBy == nil || *items[0].ApprovedBy != 99 || items[0].Status != "APPROVED" {
		t.Fatalf("items=%+v", items)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
