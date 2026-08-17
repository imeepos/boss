package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// ReleasePortByOrder(端口释放)单测:取消/超时回滚预占,只回收挂在本订单上的 RESERVED 端口。
func TestPGStore_ReleasePortByOrder(t *testing.T) {
	t.Run("有预占端口则释放", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`UPDATE ports SET status = 'IDLE', order_id = NULL`).
			WithArgs(int64(42)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock)
		if err := s.ReleasePortByOrder(context.Background(), 42); err != nil {
			t.Fatalf("ReleasePortByOrder: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("无预占端口也不报错(幂等)", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`UPDATE ports SET status = 'IDLE', order_id = NULL`).
			WithArgs(int64(7)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		s := NewPGStore(mock)
		if err := s.ReleasePortByOrder(context.Background(), 7); !errors.Is(err, ErrPortNotAvailable) {
			t.Fatalf("err=%v, want ErrPortNotAvailable", err)
		}
	})
}
