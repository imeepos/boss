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

	mock.ExpectBegin() // advance 事务;守卫拒绝后回滚
	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-7"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(2)).
		WillReturnRows(mock.NewRows([]string{"result"}).AddRow("PENDING"))
	mock.ExpectRollback()

	s := NewPGStore(mock, stubExists{})
	if err := s.Reserve(context.Background(), 7); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("err=%v, want ErrIllegalTransition", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestAdvance_AppendFailureRollsBackCounter 回归(持久化整改):计数器与环节日志必须同事务。
// order_stages 写入失败时 orders.stage 的前移一并回滚——
// 否则进程重试 advance 时恒撞 ErrIllegalTransition(计数器已到 N 而日志缺 N),订单永久卡死。
func TestAdvance_AppendFailureRollsBackCounter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-7"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(2)).
		WillReturnRows(mock.NewRows([]string{"result"}).AddRow("DONE"))
	mock.ExpectExec(`UPDATE orders SET stage`).
		WithArgs(int64(7), int8(3), "RESERVED").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO order_stages`).
		WithArgs(int64(7), int8(3), "DONE").
		WillReturnError(errors.New("db connection reset"))
	mock.ExpectRollback()

	s := NewPGStore(mock, stubExists{})
	if err := s.Reserve(context.Background(), 7); err == nil {
		t.Fatal("want error when stage log insert fails")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestAdvance_CommitFailureReturnsError 回归:提交失败必须报错并回滚,
// 不允许调用方拿到"推进成功"假象(留痕缺失即视为推进失败)。
func TestAdvance_CommitFailureReturnsError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-7"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(2)).
		WillReturnRows(mock.NewRows([]string{"result"}).AddRow("DONE"))
	mock.ExpectExec(`UPDATE orders SET stage`).
		WithArgs(int64(7), int8(3), "RESERVED").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO order_stages`).
		WithArgs(int64(7), int8(3), "DONE").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit: broken pipe"))

	s := NewPGStore(mock, stubExists{})
	if err := s.Reserve(context.Background(), 7); err == nil {
		t.Fatal("want error when commit fails")
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

	mock.ExpectBegin() // advance 事务;前置日志缺失拒绝后回滚
	mock.ExpectQuery(`SELECT stage, status, order_no FROM orders`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"stage", "status", "order_no"}).AddRow(int8(2), "PENDING", "ORD-7"))
	mock.ExpectQuery(`SELECT result FROM order_stages WHERE order_id=\$1 AND stage=\$2`).
		WithArgs(int64(7), int8(2)).
		WillReturnError(pgx.ErrNoRows)
	mock.ExpectRollback()

	s := NewPGStore(mock, stubExists{})
	if err := s.Reserve(context.Background(), 7); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("err=%v, want ErrIllegalTransition", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
