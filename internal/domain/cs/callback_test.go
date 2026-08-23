package cs

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGKnowledgeStoreCreateCallback(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	when := time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)
	pool.ExpectQuery(`INSERT INTO cs_callbacks`).WithArgs(int64(11), int64(22), when, "", int16(0), "", int64(0)).WillReturnRows(pgmockRows(9))
	store := NewPGKnowledgeStore(pool)
	id, err := store.CreateCallback(context.Background(), Callback{TicketID: 11, CustomerID: 22, ScheduledAt: when})
	if err != nil {
		t.Fatal(err)
	}
	if id != 9 {
		t.Fatalf("id=%d", id)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func pgmockRows(id int64) *pgxmock.Rows {
	return pgxmock.NewRows([]string{"id"}).AddRow(id)
}
