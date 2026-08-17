package asset

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// HandleStocktakeDiff 盘点差异项处理(TDD 先行):处理完成后任务置 DONE。
func TestPGStore_HandleStocktakeDiff(t *testing.T) {
	t.Run("DOING→DONE", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE stocktakes SET status = 'DONE'`).
			WithArgs(int64(5)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.HandleStocktakeDiff(context.Background(), 5); err != nil {
			t.Fatalf("HandleStocktakeDiff: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("任务不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE stocktakes`).
			WithArgs(int64(99)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.HandleStocktakeDiff(context.Background(), 99); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
