package billing

// Q3 账务边界测试集:锁定预付费排除、账期幂等、收款幂等与状态原子迁移的 SQL 契约。
// 边界口径:预付费不进月度出账;同客户同账期唯一;pay_no 唯一;开票同账期不重复;
// 缴费置 PAID 仅从 UNPAID 迁移(重复缴费不改已付账单)。退款边界由 feat/q3-payment-refund 承接。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 边界1:预付费客户不进月度出账(POSTPAID 过滤必须出现在出账 SQL)。
func TestGenerateBills_ExcludesPrepaid(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`billing_mode = 'POSTPAID'`).
		WithArgs("2026-09").
		WillReturnResult(pgxmock.NewResult("INSERT", 2))
	n, err := NewPGStore(mock).GenerateBills(context.Background(), "2026-09")
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("prepaid filter missing: %v", err)
	}
}

// 边界2:同客户同账期幂等(重复出账不产生新账单)。
func TestGenerateBills_PeriodIdempotent(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`ON CONFLICT \(customer_id, period\) DO NOTHING`).
		WithArgs("2026-09").
		WillReturnResult(pgxmock.NewResult("INSERT", 0))
	n, err := NewPGStore(mock).GenerateBills(context.Background(), "2026-09")
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("period uniqueness missing: %v", err)
	}
}

// 边界3:缴费置 PAID 从 UNPAID/OVERDUE 迁移;已 PAID 不重复改账单状态;customer_id 双挂回填。
func TestRecordPayment_BillPaidTransitionGuard(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT customer_id FROM bills`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(7)))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-B1", int64(1), int64(7), 100.0, "cash", "SUCCESS").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(11)))
	mock.ExpectExec(`UPDATE bills SET status = 'PAID' WHERE id = .* AND status IN \('UNPAID','OVERDUE'\)`).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectCommit()
	r, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
		Payment{PayNo: "PAY-B1", BillID: 1, Amount: 100.0, Method: "cash"})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if r.PaymentID != 11 {
		t.Fatalf("receipt=%+v", r)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("UNPAID guard missing: %v", err)
	}
}

// 边界4:FAILED 缴费不置 PAID;customer_id 同样双挂回填。
func TestRecordPayment_FailedNotMarkPaid(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT customer_id FROM bills`).WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(8)))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-B2", int64(2), int64(8), 50.0, "card", "FAILED").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(12)))
	// 无 UPDATE bills 期望:出现即 ExpectationsWereMet 失败
	mock.ExpectCommit()
	if _, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
		Payment{PayNo: "PAY-B2", BillID: 2, Amount: 50.0, Method: "card", Status: "FAILED"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("FAILED must not mark paid: %v", err)
	}
}

// 边界6:落账锚定门禁——账单不存在拒收;bill_id/customer_id 双空拒收;双挂不一致拒收。
func TestRecordPayment_RejectsOrphanPayment(t *testing.T) {
	t.Run("账单不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT customer_id FROM bills`).WithArgs(int64(999)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}))
		_, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
			Payment{PayNo: "PAY-X", BillID: 999, Amount: 10, Method: "cash"})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
	t.Run("双挂不一致", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT customer_id FROM bills`).WithArgs(int64(1)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(7)))
		_, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
			Payment{PayNo: "PAY-X", BillID: 1, CustomerID: 99, Amount: 10, Method: "cash"})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
	t.Run("bill_id 与 customer_id 双空", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		_, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
			Payment{PayNo: "PAY-X", Amount: 10, Method: "cash"})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})
}

// 边界5:开票同账期不重复(NOT EXISTS 在发票;重跑出账不重复开票)。
func TestIssueInvoices_PeriodNoDuplicate(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`NOT EXISTS`).
		WithArgs("2026-09").
		WillReturnRows(mock.NewRows([]string{"id"})) // 全部已开票 → 空输入
	res, err := NewPGStore(mock).IssueInvoicesForPeriod(context.Background(), "2026-09")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if res.Issued != 0 || len(res.FailedIDs) != 0 {
		t.Fatalf("res=%+v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("no-duplicate predicate missing: %v", err)
	}
}
