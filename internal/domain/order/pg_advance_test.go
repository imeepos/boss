package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestAdvance_BlockedWhenPrevStagePending 回归(2026-08-25 审计 §2.3.2):
// 环节2 PENDING(资源不可用)时,即使 orders.stage 计数器已到 2,也不得推进环节3。
// 此前 advance 只看计数器,产生"引用有效但环节乱序"悬案(订单 335/336/337/377/383)。
func TestAdvance_BlockedWhenPrevStagePending(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-7"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(2)).
		WillReturnRows(mock.NewRows([]string{"result"}).AddRow("PENDING"))

	s := NewPGStore(mock, stubExists{})
	if err := s.Reserve(context.Background(), 7); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("err=%v, want ErrIllegalTransition", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestAdvance_BlockedWhenPrevStageMissing 回归:前置环节日志行缺失(计数器与日志分叉)同样拒绝。
func TestAdvance_BlockedWhenPrevStageMissing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-7"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(2)).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock, stubExists{})
	if err := s.Reserve(context.Background(), 7); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("err=%v, want ErrIllegalTransition", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
