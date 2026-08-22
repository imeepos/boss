package quadlink

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// ClearanceStats(4 小时清零率,000114)形状回归 + 比率计算。
func TestPGStore_ClearanceStats(t *testing.T) {
	t.Run("有事件按占比计算", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"a", "b", "c"}).AddRow(int64(20), int64(19), int64(1)))

		s := NewPGStore(mock)
		st, err := s.ClearanceStats(context.Background())
		if err != nil {
			t.Fatalf("ClearanceStats: %v", err)
		}
		if st.Discovered != 20 || st.ClearedWithin4h != 19 || st.StillOpen != 1 {
			t.Fatalf("st=%+v", st)
		}
		if st.Rate < 0.9499 || st.Rate > 0.9501 {
			t.Fatalf("rate=%v, want 0.95", st.Rate)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("无事件 rate=1", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`FROM quad_links`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"a", "b", "c"}).AddRow(int64(0), int64(0), int64(0)))

		s := NewPGStore(mock)
		st, err := s.ClearanceStats(context.Background())
		if err != nil {
			t.Fatalf("ClearanceStats: %v", err)
		}
		if st.Rate != 1 {
			t.Fatalf("rate=%v, want 1", st.Rate)
		}
	})
}

// ResolveConflict/Reconcile 时间线维护形状回归:cleared_at 留痕。
func TestPGStore_ConflictTimelineSQL(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE quad_links SET status = 'UNLINKED', cleared_at = now\(\)`).
		WithArgs(int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.ResolveConflict(context.Background(), 5); err != nil {
		t.Fatalf("ResolveConflict: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
