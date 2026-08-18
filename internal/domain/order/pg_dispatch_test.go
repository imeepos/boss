package order

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_AssignDispatchTicket 契约:指派回填师傅;未命中返回 ErrOrderNotFound。
func TestPGStore_AssignDispatchTicket(t *testing.T) {
	t.Run("指派成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE dispatch_tickets SET worker_id=\$2, worker_name=\$3 WHERE ticket_no=\$1`).
			WithArgs("TK-1", int64(5), "张师傅").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock, nil, nil).AssignDispatchTicket(context.Background(), "TK-1", 5, "张师傅"); err != nil {
			t.Fatalf("AssignDispatchTicket: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE dispatch_tickets`).WithArgs("TK-X", int64(5), "张师傅").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		err := NewPGStore(mock, nil, nil).AssignDispatchTicket(context.Background(), "TK-X", 5, "张师傅")
		if !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
