// 回归:PG Emit 复活语义 —— 同键已存在(含已办结)时走 UPDATE 分支刷新为未办并清读回执。
package notify

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_EmitInsertsWhenAbsent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer mock.Close()
	mock.ExpectExec(`WITH revived AS \(\s*UPDATE admin_notifications`).
		WithArgs("todo", "WARN", "t", "", "", "realname", "customer/1", "", nil).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	s := NewPGStore(mock)
	if err := s.Emit(context.Background(), Input{Category: CategoryTodo, Title: "t", RefType: "realname", RefID: "customer/1"}); err != nil {
		t.Fatalf("Emit insert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPGStore_EmitInvalid(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer mock.Close()
	s := NewPGStore(mock)
	if err := s.Emit(context.Background(), Input{Category: "spam", Title: "x", RefType: "r", RefID: "1"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}
