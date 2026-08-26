package billing

// RecordTopup 充值落账契约:流水+余额同事务;FAILED 不动余额;孤儿/带账单拒绝;唯一冲突幂等。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

// pgErrUnique23505 唯一约束冲突(pg 23505)桩。
var pgErrUnique23505 = &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}

// 成功充值:插入无账单流水 + portal_wallets 余额原子增加。
func TestRecordTopup_SuccessCreditsBalance(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-T1", int64(7), 50.0, "card", "SUCCESS").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(21)))
	mock.ExpectExec(`INSERT INTO portal_wallets`).
		WithArgs(int64(7), 50.0).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	id, err := NewPGStore(mock).RecordTopup(context.Background(),
		Payment{PayNo: "PAY-T1", CustomerID: 7, Amount: 50, Method: "card", Status: "SUCCESS"})
	if err != nil {
		t.Fatalf("topup: %v", err)
	}
	if id != 21 {
		t.Fatalf("id=%d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("balance credit missing: %v", err)
	}
}

// 失败充值:只落 FAILED 流水,不动余额(无 portal_wallets 期望即验证)。
func TestRecordTopup_FailedNoBalance(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-T2", int64(7), 50.0, "card", "FAILED").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(22)))
	mock.ExpectCommit()
	if _, err := NewPGStore(mock).RecordTopup(context.Background(),
		Payment{PayNo: "PAY-T2", CustomerID: 7, Amount: 50, Method: "card", Status: "FAILED"}); err != nil {
		t.Fatalf("topup: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("FAILED must not touch balance: %v", err)
	}
}

// 锚定门禁:带账单拒绝;customer_id 为空拒绝(防孤儿流水)。
func TestRecordTopup_AnchorGuard(t *testing.T) {
	t.Run("带账单拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		_, err := NewPGStore(mock).RecordTopup(context.Background(),
			Payment{PayNo: "PAY-X", BillID: 1, CustomerID: 7, Amount: 10, Method: "cash"})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
	t.Run("customer_id 为空拒绝", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		_, err := NewPGStore(mock).RecordTopup(context.Background(),
			Payment{PayNo: "PAY-X", Amount: 10, Method: "cash"})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
}

// 唯一冲突:同 pay_no 已落账 → ErrPayNoExists(调用方按幂等处理,渠道重投不重复到账)。
func TestRecordTopup_PayNoUniqueViolation(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-DUP", int64(7), 10.0, "card", "SUCCESS").
		WillReturnError(pgErrUnique23505)
	if _, err := NewPGStore(mock).RecordTopup(context.Background(),
		Payment{PayNo: "PAY-DUP", CustomerID: 7, Amount: 10, Method: "card", Status: "SUCCESS"}); !errors.Is(err, ErrPayNoExists) {
		t.Fatalf("err=%v, want ErrPayNoExists", err)
	}
}
