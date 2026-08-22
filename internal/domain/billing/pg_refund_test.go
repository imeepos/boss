package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// expectRefundTx 桩一次成功全额退款事务:锁流水→置 REFUNDED→账单回 UNPAID→提交→回读。
func expectRefundTx(t *testing.T, m pgxmock.PgxPoolIface, paymentID, billID int64) {
	t.Helper()
	m.ExpectBegin()
	m.ExpectQuery(`SELECT bill_id FROM payments WHERE id = \$1 FOR UPDATE`).WithArgs(paymentID).
		WillReturnRows(m.NewRows([]string{"bill_id"}).AddRow(billID))
	m.ExpectExec(`UPDATE payments SET status = 'REFUNDED'`).WithArgs(paymentID, "客户申请").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	m.ExpectExec(`UPDATE bills SET status = 'UNPAID'`).WithArgs(billID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	m.ExpectCommit()
	m.ExpectQuery(`SELECT id, pay_no`).WithArgs(paymentID).
		WillReturnRows(m.NewRows([]string{"id", "pay_no", "bill_id", "amount", "method",
			"status", "refund_reason", "refunded_at"}).
			AddRow(paymentID, "PAY-20260820-001", billID, 999.0, "wechat",
				"REFUNDED", "客户申请", time.Now()))
}

// TestPGStore_RefundPayment 契约(000112):全额退款留痕 + 账单回 UNPAID。
func TestPGStore_RefundPayment(t *testing.T) {
	t.Run("全额退款", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectRefundTx(t, mock, 3, 1)

		p, err := NewPGStore(mock).RefundPayment(context.Background(), 3, "客户申请")
		if err != nil {
			t.Fatalf("refund: %v", err)
		}
		if p.Status != "REFUNDED" || p.RefundReason != "客户申请" || p.RefundedAt == nil {
			t.Fatalf("p=%+v", p)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("充值流水退款不动账单", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`FOR UPDATE`).WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"bill_id"}).AddRow(nil))
		mock.ExpectExec(`UPDATE payments SET status = 'REFUNDED'`).WithArgs(int64(5), "x").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`SELECT id, pay_no`).WithArgs(int64(5)).
			WillReturnRows(mock.NewRows([]string{"id", "pay_no", "bill_id", "amount", "method",
				"status", "refund_reason", "refunded_at"}).
				AddRow(int64(5), "PAY-R", 0, 100.0, "cash", "REFUNDED", "x", time.Now()))

		if _, err := NewPGStore(mock).RefundPayment(context.Background(), 5, "x"); err != nil {
			t.Fatalf("refund: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("重复退款被拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`FOR UPDATE`).WithArgs(int64(3)).
			WillReturnRows(mock.NewRows([]string{"bill_id"}).AddRow(int64(1)))
		mock.ExpectExec(`UPDATE payments SET status = 'REFUNDED'`).WithArgs(int64(3), "again").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectRollback()

		_, err := NewPGStore(mock).RefundPayment(context.Background(), 3, "again")
		if !errors.Is(err, ErrPaymentNotRefundable) {
			t.Fatalf("err=%v, want ErrPaymentNotRefundable", err)
		}
	})
	t.Run("流水不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectBegin()
		mock.ExpectQuery(`FOR UPDATE`).WithArgs(int64(99)).
			WillReturnError(errors.New("no rows in result set"))
		mock.ExpectRollback()

		if _, err := NewPGStore(mock).RefundPayment(context.Background(), 99, "x"); err == nil {
			t.Fatal("want error")
		}
	})
}
