package billing

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_MarkOverdueBills(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`UPDATE bills SET status = 'OVERDUE'`).WithArgs(15).
		WillReturnResult(pgxmock.NewResult("UPDATE", 7))
	n, err := NewPGStore(mock).MarkOverdueBills(context.Background(), 15)
	if err != nil || n != 7 {
		t.Fatalf("n=%d err=%v, want 7/nil", n, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListOverdueCustomers(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	oldest := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`FROM bills WHERE status = 'OVERDUE'`).WillReturnRows(
		mock.NewRows([]string{"customer_id", "sum", "min"}).
			AddRow(int64(213), 1998.0, oldest))
	got, err := NewPGStore(mock).ListOverdueCustomers(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if got[0].CustomerID != 213 || got[0].Amount != 1998.0 || !got[0].OldestAt.Equal(oldest) {
		t.Fatalf("got=%+v", got[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestArrearsDaysFrom(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		oldest time.Time
		want   int32
	}{{now.AddDate(0, 0, -30), 30}, {now.Add(2 * time.Hour), 0}, {now.Add(-2 * time.Hour), 0}}
	for _, c := range cases {
		if got := ArrearsDaysFrom(now, c.oldest); got != c.want {
			t.Fatalf("ArrearsDaysFrom(%v)=%d, want %d", c.oldest, got, c.want)
		}
	}
}
