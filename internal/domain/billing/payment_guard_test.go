package billing

// 柜面收款边界测试(纪要 2026-08-28-柜面现金收款):
// method 白名单、pay_no 兜底生成、条件置 PAID(PARTIAL 口径)、柜台日结汇总/回填。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 渠道→资金通道映射(terms.md 柜面裁定 2026-08-28):各渠道只取本通道流水,未知渠道全量兜底。
func TestChannelMethods(t *testing.T) {
	cases := map[string][]string{
		"微信":    {"wechat"},
		"支付宝":   {"alipay"},
		"线下营业厅": {"cash", "offline"},
		"柜面收单":  {"card"},
		"自定义渠道": nil,
	}
	for ch, want := range cases {
		got := channelMethods(ch)
		if len(got) != len(want) {
			t.Fatalf("%s: %v, want %v", ch, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: %v, want %v", ch, got, want)
			}
		}
	}
}

// 边界7:method 白名单外拒收(脏 method 毁渠道对账);拒收发生在任何 SQL 之前。
func TestRecordPayment_InvalidMethodRejected(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	_, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
		Payment{PayNo: "PAY-M1", CustomerID: 7, Amount: 10, Method: "gold"})
	if !errors.Is(err, ErrInvalidMethod) {
		t.Fatalf("err=%v, want ErrInvalidMethod", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("rejected payment must not touch db: %v", err)
	}
}

// 边界8:pay_no 未传时兜底生成(防空值撞唯一约束);格式 PAY-<14位时间戳>-<4位hex>。
func TestRecordPayment_GeneratesPayNoWhenEmpty(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT customer_id FROM bills`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(7)))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs(pgxmock.AnyArg(), int64(1), int64(7), 10.0, "cash", "SUCCESS", "", "", "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(21)))
	mock.ExpectExec(`UPDATE bills SET status = 'PAID'`).WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	if _, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
		Payment{BillID: 1, Amount: 10, Method: "cash"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	no := genPayNo()
	if !strings.HasPrefix(no, "PAY-") || len(strings.Split(no, "-")) != 3 {
		t.Fatalf("genPayNo=%s, want PAY-<ts>-<hex>", no)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("pay_no fallback: %v", err)
	}
}

// 边界9:部分收款——置 PAID 必须带"累计实收≥应收"条件谓词;谓词不满足(0 行更新)仍正常提交。
func TestRecordPayment_PartialNotMarkPaid(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT customer_id FROM bills`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(7)))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).
		WithArgs("PAY-P1", int64(1), int64(7), 50.0, "cash", "SUCCESS", "旗舰店", "01", "alice").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(31)))
	// 应收 100 收 50:谓词含累计实收子查询,DB 侧判 0 行,提交照常。
	mock.ExpectExec(`UPDATE bills SET status = 'PAID' WHERE id = \$1 AND status IN \('UNPAID','OVERDUE'\)\s+AND \(COALESCE\(\(SELECT SUM\(amount\) FROM payments`).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectCommit()
	r, err := NewPGStore(mock).RecordPaymentWithCoupon(context.Background(),
		Payment{PayNo: "PAY-P1", BillID: 1, Amount: 50.0, Method: "cash", SiteName: "旗舰店", CounterCode: "01", OperatorName: "alice"})
	if err != nil || r.PaymentID != 31 {
		t.Fatalf("r=%+v err=%v", r, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("partial predicate missing: %v", err)
	}
}

// 边界10:日结汇总收入/退款分列,净额=收入-退款,回填实点按网点+操作员合并。
func TestDailyCashSummary(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	date := "2026-08-28"
	mock.ExpectQuery(`SUM\(CASE WHEN status = 'SUCCESS' THEN amount END\)`).WithArgs(date).
		WillReturnRows(mock.NewRows([]string{"site_name", "operator_name", "in", "refund"}).
			AddRow("旗舰店", "alice", 800.0, 100.0).
			AddRow("旗舰店", "bob", 300.0, 0.0))
	mock.ExpectQuery(`FROM payment_daily_closings WHERE closing_date`).WithArgs(date).
		WillReturnRows(mock.NewRows([]string{"site_name", "operator_name", "counted_amount"}).
			AddRow("旗舰店", "alice", 700.0))
	rows, err := NewPGStore(mock).DailyCashSummary(context.Background(), date)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(rows))
	}
	a := rows[0]
	if a.InAmount != 800 || a.RefundAmount != 100 || a.NetAmount != 700 {
		t.Fatalf("alice row=%+v", a)
	}
	if a.CountedAmount == nil || *a.CountedAmount != 700 {
		t.Fatalf("alice counted=%v", a.CountedAmount)
	}
	if rows[1].CountedAmount != nil {
		t.Fatalf("bob counted should be nil")
	}
}

// 边界11:实点回填 UPSERT 返回系统净额快照与差异;|diff|≤0.005 记平账。
func TestSaveDailyClosing(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SUM\(CASE WHEN status = 'SUCCESS' THEN amount ELSE -amount END\)`).
		WithArgs("2026-08-28", "旗舰店", "alice").
		WillReturnRows(mock.NewRows([]string{"net"}).AddRow(700.0))
	mock.ExpectQuery(`ON CONFLICT \(closing_date, site_name, operator_name\)`).
		WithArgs("2026-08-28", "旗舰店", "alice", 700.0, 699.5, "bob").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(5)))
	res, err := NewPGStore(mock).SaveDailyClosing(context.Background(), DailyClosing{
		Date: "2026-08-28", SiteName: "旗舰店", OperatorName: "alice", CountedAmount: 699.5, CreatedBy: "bob",
	})
	if err != nil {
		t.Fatalf("closing: %v", err)
	}
	if res.ID != 5 || res.SystemAmount != 700 || res.DiffAmount != 0.5 || res.Balanced {
		t.Fatalf("res=%+v", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("upsert missing: %v", err)
	}
}

// 边界12:日结下钻逐笔含退款流水(收入/退款分列勾对的明细面)。
func TestCashPaymentsByDate(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`FROM payments\s+WHERE method = 'cash' AND created_at::date`).
		WithArgs("2026-08-28").
		WillReturnRows(mock.NewRows([]string{"id", "pay_no", "bill_id", "customer_id",
			"amount", "status", "refund_reason", "refunded_at", "site_name", "counter_code", "operator_name"}).
			AddRow(int64(1), "PAY-D1", nil, int64(7), 100.0, "REFUNDED", "错收", nil, "旗舰店", "01", "alice"))
	items, err := NewPGStore(mock).CashPaymentsByDate(context.Background(), "2026-08-28")
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].RefundReason != "错收" || items[0].SiteName != "旗舰店" {
		t.Fatalf("item=%+v", items[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("items query: %v", err)
	}
}
