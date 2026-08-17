package quadlink

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// ResolveConflict 四码冲突处理(人工修复后置回 UNLINKED 待重绑)。
func TestPGStore_ResolveConflict(t *testing.T) {
	t.Run("CONFLICT→UNLINKED", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectExec(`UPDATE quad_links SET status = 'UNLINKED'`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock)
		if err := s.ResolveConflict(context.Background(), 1); err != nil {
			t.Fatalf("ResolveConflict: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("非冲突态拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()

		mock.ExpectExec(`UPDATE quad_links SET status = 'UNLINKED'`).
			WithArgs(int64(2)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		s := NewPGStore(mock)
		err := s.ResolveConflict(context.Background(), 2)
		if !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v, want ErrIllegalTransition", err)
		}
	})
}
