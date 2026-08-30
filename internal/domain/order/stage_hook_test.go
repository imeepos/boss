// advance 的 order.stage.done 广播测试(Q4 开放平台 M5 业务事件挂接)。
package order

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// recordingNotifier 记录 Emit 调用。
type recordingNotifier struct {
	calls []struct {
		eventType string
		eventID   string
	}
	err error
}

func (r *recordingNotifier) Emit(_ context.Context, eventType, eventID string, _ any) (int, error) {
	r.calls = append(r.calls, struct {
		eventType string
		eventID   string
	}{eventType, eventID})
	return len(r.calls), r.err
}

// newAdvanceMock 准备 PreConfigOLT 的 SQL 期望(环节7,stage 6→7)。
func newAdvanceMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT customer_id, legal_entity_id FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"customer_id", "legal_entity_id"}).AddRow(int64(1), int64(1)))
	mock.ExpectBegin() // advance 事务:计数器+环节日志原子落库
	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(6), "RESERVED", "ORD-20250817-001"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(6)).
		WillReturnRows(mock.NewRows([]string{"result"}).AddRow("DONE"))
	mock.ExpectExec(`UPDATE orders SET stage`).
		WithArgs(int64(7), int8(7), "RESERVED").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO order_stages`).
		WithArgs(int64(7), int8(7), "DONE").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	return mock
}

func TestAdvanceEmitsStageDone(t *testing.T) {
	mock := newAdvanceMock(t)
	defer mock.Close()

	s := NewPGStore(mock, stubExists{ok: true}, &stubProfileCreator{}, &stubProvCreator{})
	n := &recordingNotifier{}
	s.SetStageNotifier(n)

	if err := s.PreConfigOLT(context.Background(), 7); err != nil {
		t.Fatalf("ApplyTag: %v", err)
	}
	if len(n.calls) != 1 {
		t.Fatalf("want 1 emit, got %d", len(n.calls))
	}
	if n.calls[0].eventType != StageEventType || n.calls[0].eventID != "ORD-20250817-001:stage:7" {
		t.Fatalf("unexpected emit: %+v", n.calls[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestAdvanceWithoutNotifierStillWorks(t *testing.T) {
	mock := newAdvanceMock(t)
	defer mock.Close()

	s := NewPGStore(mock, stubExists{ok: true}, &stubProfileCreator{}, &stubProvCreator{})
	if err := s.PreConfigOLT(context.Background(), 7); err != nil {
		t.Fatalf("ApplyTag without notifier: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestAdvanceEmitErrorDoesNotFailStep(t *testing.T) {
	mock := newAdvanceMock(t)
	defer mock.Close()

	s := NewPGStore(mock, stubExists{ok: true}, &stubProfileCreator{}, &stubProvCreator{})
	s.SetStageNotifier(&recordingNotifier{err: context.Canceled}) // Emit 失败
	if err := s.PreConfigOLT(context.Background(), 7); err != nil {
		t.Fatalf("emit error must not fail advance: %v", err)
	}
}
