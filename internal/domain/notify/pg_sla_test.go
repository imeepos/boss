package notify

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// Emit due_at 落库 + EscalateOverdue/SLAStats(000115 P1 时限)形状回归。
func TestPGStore_SLA(t *testing.T) {
	t.Run("todo 带 DueHours 写 due_at", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`INSERT INTO admin_notifications`).
			WithArgs("todo", "WARN", "T", "", "", "rt", "1", "", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock)
		if err := s.Emit(context.Background(), Input{
			Category: CategoryTodo, Title: "T", RefType: "rt", RefID: "1", DueHours: 4,
		}); err != nil {
			t.Fatalf("Emit: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("task 不带 due_at(NULL)", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`INSERT INTO admin_notifications`).
			WithArgs("task", "INFO", "T", "", "", "rt", "2", "", nil).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		s := NewPGStore(mock)
		if err := s.Emit(context.Background(), Input{
			Category: CategoryTask, Title: "T", RefType: "rt", RefID: "2", DueHours: 4,
		}); err != nil {
			t.Fatalf("Emit: %v", err)
		}
	})
	t.Run("EscalateOverdue 只升超时未办", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`UPDATE admin_notifications SET level = 'URGENT'`).
			WillReturnResult(pgxmock.NewResult("UPDATE", 2))

		s := NewPGStore(mock)
		n, err := s.EscalateOverdue(context.Background())
		if err != nil || n != 2 {
			t.Fatalf("n=%d err=%v", n, err)
		}
	})
	t.Run("SLAStats 比率计算", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`FROM admin_notifications`).
			WithArgs("7").
			WillReturnRows(pgxmock.NewRows([]string{"a", "b", "c", "d"}).
				AddRow(int64(20), int64(19), int64(0), int64(1)))

		s := NewPGStore(mock)
		st, err := s.SLAStats(context.Background(), 7)
		if err != nil {
			t.Fatalf("SLAStats: %v", err)
		}
		if st.WithDeadline != 20 || st.ResolvedOnTime != 19 || st.OverdueOpen != 1 {
			t.Fatalf("st=%+v", st)
		}
		if st.OnTimeRate < 0.9499 || st.OnTimeRate > 0.9501 {
			t.Fatalf("rate=%v", st.OnTimeRate)
		}
	})
	t.Run("SLAStats 无样本 rate=1", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`FROM admin_notifications`).
			WithArgs("7").
			WillReturnRows(pgxmock.NewRows([]string{"a", "b", "c", "d"}).
				AddRow(int64(0), int64(0), int64(0), int64(0)))

		s := NewPGStore(mock)
		st, err := s.SLAStats(context.Background(), 7)
		if err != nil || st.OnTimeRate != 1 {
			t.Fatalf("st=%+v err=%v", st, err)
		}
	})
}

// MemStore 时限字段同步:DueHours→DueAt,Resolve→ResolvedAt。
func TestMemStore_SLAFields(t *testing.T) {
	s := NewMemStore()
	if err := s.Emit(context.Background(), Input{
		Category: CategoryTodo, Title: "T", RefType: "rt", RefID: "1", DueHours: 1,
	}); err != nil {
		t.Fatal(err)
	}
	items, _, _ := s.List(context.Background(), "sysadmin", 1, Filter{})
	if items[0].DueAt == "" {
		t.Fatalf("DueAt should be set: %+v", items[0])
	}
	if err := s.Resolve(context.Background(), "rt", "1"); err != nil {
		t.Fatal(err)
	}
	items, _, _ = s.List(context.Background(), "sysadmin", 1, Filter{})
	if !items[0].Resolved || items[0].ResolvedAt == "" {
		t.Fatalf("ResolvedAt should be set: %+v", items[0])
	}
}
