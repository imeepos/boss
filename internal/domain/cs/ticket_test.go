package cs

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func mustTime() time.Time {
	return time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
}

func TestTransitionTicket_ToProcessing(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`WITH old AS`).
		WithArgs("TKT-1", Processing, int64(7), "take over").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	store := NewPGKnowledgeStore(pool)
	if err := store.TransitionTicket(context.Background(), "TKT-1", Processing, 7, "take over"); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionTicket_ToClosed(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`WITH old AS`).
		WithArgs("TKT-2", Closed, int64(0), "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	store := NewPGKnowledgeStore(pool)
	if err := store.TransitionTicket(context.Background(), "TKT-2", Closed, 0, ""); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTransitionTicket_InvalidStatus(t *testing.T) {
	store := NewPGKnowledgeStore(nil)
	if err := store.TransitionTicket(context.Background(), "TKT-3", "BOGUS", 0, ""); err == nil {
		t.Fatal("want error for invalid status")
	}
}

func TestTransitionTicket_NotFound(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`WITH old AS`).
		WithArgs("TKT-X", Processing, int64(0), "").
		WillReturnResult(pgxmock.NewResult("INSERT", 0))

	store := NewPGKnowledgeStore(pool)
	if err := store.TransitionTicket(context.Background(), "TKT-X", Processing, 0, ""); err == nil {
		t.Fatal("want error for missing ticket")
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEscalateTicket(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectExec(`WITH changed AS`).
		WithArgs("TKT-1", int64(9), "SLA overdue").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	store := NewPGKnowledgeStore(pool)
	if err := store.EscalateTicket(context.Background(), "TKT-1", 9, "SLA overdue"); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListTicketEventsByNo(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	pool.ExpectQuery(`SELECT e.id, e.ticket_id, e.event_type`).
		WithArgs("TKT-1").
		WillReturnRows(pool.NewRows([]string{"id", "ticket_id", "event_type", "from_status", "to_status", "actor_id", "note", "created_at"}).
			AddRow(int64(1), int64(11), "STATUS_CHANGED", "OPEN", "PROCESSING", int64(7), "take over", mustTime()))

	store := NewPGKnowledgeStore(pool)
	events, err := store.ListTicketEventsByNo(context.Background(), "TKT-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventType != "STATUS_CHANGED" || events[0].FromStatus != "OPEN" || events[0].ToStatus != "PROCESSING" {
		t.Fatalf("events=%+v", events)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteCallback(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	when := mustTime()
	pool.ExpectExec(`UPDATE cs_callbacks SET completed_at=`).
		WithArgs(int64(3), &when, "SATISFIED", int16(5), "all good", int64(8)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	store := NewPGKnowledgeStore(pool)
	if err := store.CompleteCallback(context.Background(), 3, Callback{
		CompletedAt: &when, Result: "SATISFIED", Rating: 5, Comment: "all good", OperatorID: 8,
	}); err != nil {
		t.Fatal(err)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
