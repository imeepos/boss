package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// expectIssueTx 桩一次成功开票事务:查重(空)→账单(联法人属地)→占号→插入→提交。
// 金额契约 TAX-TC1:净额 999 + VAT 12% 119.88 = 合计 1118.88,编号 INV-00000001;
// 属地契约(000109):法人 CN/manual 快照落票。
func expectIssueTx(t *testing.T, mock pgxmock.PgxPoolIface, billID int64) {
	t.Helper()
	mock.ExpectQuery(`SELECT 1 FROM invoices`).WithArgs(billID).
		WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery(`FROM bills b LEFT JOIN legal_entities`).WithArgs(billID).
		WillReturnRows(mock.NewRows([]string{"id", "bill_no", "customer_id", "customer_name",
			"amount", "tax_jurisdiction", "tax_channel"}).
			AddRow(billID, "BILL-202608-201", int64(1), "王先生", 999.0, "CN", "manual"))
	mock.ExpectQuery(`UPDATE arn_sequences`).WithArgs(arnDocType).
		WillReturnRows(mock.NewRows([]string{"next_no", "prefix"}).AddRow(int64(1), "INV-"))
	mock.ExpectQuery(`INSERT INTO invoices`).
		WithArgs("INV-00000001", billID, "BILL-202608-201", int64(1), "王先生", "王先生",
			999.0, VATRate, 119.88, 1118.88, "CN", "manual", "PENDING").
		WillReturnRows(mock.NewRows([]string{"id", "issued_at"}).AddRow(int64(11), time.Now()))
}

func TestPGStore_IssueInvoiceForBill_TaxTC1(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	expectIssueTx(t, mock, 1)
	mock.ExpectCommit()

	inv, err := NewPGStore(mock).issueInvoiceForBillByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if inv.InvoiceNo != "INV-00000001" {
		t.Fatalf("invoiceNo=%s, want INV-00000001", inv.InvoiceNo)
	}
	if inv.NetAmount != 999.0 || inv.VatAmount != 119.88 || inv.TotalAmount != 1118.88 {
		t.Fatalf("net=%v vat=%v total=%v, want 999/119.88/1118.88", inv.NetAmount, inv.VatAmount, inv.TotalAmount)
	}
	if inv.Status != "ISSUED" || inv.Title != "王先生" {
		t.Fatalf("inv=%+v", inv)
	}
	if inv.TaxJurisdiction != "CN" || inv.TaxChannel != "manual" || inv.TaxStatus != "PENDING" {
		t.Fatalf("tax snapshot=%+v, want CN/manual/PENDING", inv)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_IssueInvoiceForBill_Duplicate 契约(CT-007 幂等):已开票账单重复开票被拒,
// 且发生在占号之前(幂等重跑不耗 ARN,无跳号)。
func TestPGStore_IssueInvoiceForBill_Duplicate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT 1 FROM invoices`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"1"}).AddRow(int64(1)))
	mock.ExpectRollback()

	_, err = NewPGStore(mock).issueInvoiceForBillByID(context.Background(), 9)
	if !errors.Is(err, ErrDuplicateInvoice) {
		t.Fatalf("err=%v, want ErrDuplicateInvoice", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// expectLocateBill 桩 customer+billNo 定位账单(归属校验)。
func expectLocateBill(t *testing.T, mock pgxmock.PgxPoolIface, customerID int64, billNo string, billID int64) {
	t.Helper()
	mock.ExpectQuery(`SELECT id FROM bills`).WithArgs(customerID, billNo).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(billID))
}

// TestPGStore_IssueInvoiceForBillByNo 门户按单开票:定位→复用开票内核。
func TestPGStore_IssueInvoiceForBillByNo(t *testing.T) {
	t.Run("新开", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectLocateBill(t, mock, 1, "BILL-202608-201", 7)
		mock.ExpectBegin()
		expectIssueTx(t, mock, 7)
		mock.ExpectCommit()

		inv, err := NewPGStore(mock).IssueInvoiceForBill(context.Background(), 1, "BILL-202608-201")
		if err != nil {
			t.Fatalf("issue: %v", err)
		}
		if inv.InvoiceNo != "INV-00000001" || inv.ID != 11 {
			t.Fatalf("inv=%+v", inv)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("幂等返回已有票", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		expectLocateBill(t, mock, 1, "BILL-202608-201", 7)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT 1 FROM invoices`).WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"1"}).AddRow(int64(1)))
		mock.ExpectRollback()
		mock.ExpectQuery(`SELECT id, invoice_no`).WithArgs(int64(7)).WillReturnRows(taxRows(mock))

		inv, err := NewPGStore(mock).IssueInvoiceForBill(context.Background(), 1, "BILL-202608-201")
		if err != nil {
			t.Fatalf("issue: %v", err)
		}
		if inv.InvoiceNo != "INV-00000001" || inv.Status != "ISSUED" {
			t.Fatalf("inv=%+v", inv)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("不存在或不属该客户", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT id FROM bills`).WithArgs(int64(2), "NOPE").
			WillReturnError(pgx.ErrNoRows)

		if _, err := NewPGStore(mock).IssueInvoiceForBill(context.Background(), 2, "NOPE"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_VoidInvoice(t *testing.T) {
	t.Run("作废成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE invoices SET status = 'VOIDED'`).WithArgs(int64(11), "客户要求重开").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).VoidInvoice(context.Background(), 11, "客户要求重开"); err != nil {
			t.Fatalf("void: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("重复作废被拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE invoices SET status = 'VOIDED'`).WithArgs(int64(11), "again").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		err := NewPGStore(mock).VoidInvoice(context.Background(), 11, "again")
		if !errors.Is(err, ErrIllegalInvoiceTransition) {
			t.Fatalf("err=%v, want ErrIllegalInvoiceTransition", err)
		}
	})
}

// TestPGStore_RecordPayment 契约:收款流水与账单置 PAID 同事务。
func TestPGStore_RecordPayment(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO payments`).WithArgs("PAY-20260818-001", int64(1), int64(0), 999.0, "wechat", "SUCCESS").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))
	mock.ExpectExec(`UPDATE bills SET status = 'PAID'`).WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	id, err := NewPGStore(mock).RecordPayment(context.Background(), Payment{
		PayNo: "PAY-20260818-001", BillID: 1, Amount: 999.0, Method: "wechat",
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_IssueInvoicesForPeriod 契约(CT-007):出账后批量开票,单张失败不中断。
func TestPGStore_IssueInvoicesForPeriod(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT b.id FROM bills b`).WithArgs("2026-08").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)))

	// 账单1 开票成功
	mock.ExpectBegin()
	expectIssueTx(t, mock, 1)
	mock.ExpectCommit()
	// 账单2 开票失败(占号错) → 进异常清单,批次继续
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT 1 FROM invoices`).WithArgs(int64(2)).WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery(`FROM bills b LEFT JOIN legal_entities`).WithArgs(int64(2)).
		WillReturnError(errors.New("db down"))
	mock.ExpectRollback()

	res, err := NewPGStore(mock).IssueInvoicesForPeriod(context.Background(), "2026-08")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Issued != 1 || len(res.FailedIDs) != 1 || res.FailedIDs[0] != 2 {
		t.Fatalf("res=%+v, want issued=1 failed=[2]", res)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// taxCols 与 invoiceCols 对齐的行集(含 000049 税局网关列)。
func taxRows(m pgxmock.PgxPoolIface) *pgxmock.Rows {
	return m.NewRows([]string{"id", "invoice_no", "bill_id", "bill_no", "customer_id", "customer_name",
		"title", "net_amount", "vat_rate", "vat_amount", "total_amount", "status", "void_reason",
		"tax_jurisdiction", "tax_channel", "tax_status", "tax_no", "tax_fail_reason",
		"issued_at", "voided_at"}).
		AddRow(int64(11), "INV-00000001", int64(1), "BILL-202608-201", int64(1), "王先生",
			"王先生", 999.0, 0.12, 119.88, 1118.88, "ISSUED", "",
			"CN", "manual", "PENDING", "", "", time.Now(), nil)
}

func TestPGStore_GetInvoice_TaxGatewayCols(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT id, invoice_no`).WithArgs(int64(11)).WillReturnRows(taxRows(mock))
	inv, err := NewPGStore(mock).GetInvoice(context.Background(), 11)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if inv.TaxJurisdiction != "CN" || inv.TaxChannel != "manual" || inv.TaxStatus != "PENDING" || inv.TaxNo != "" {
		t.Fatalf("tax cols=%+v", inv)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_BackfillTaxNo 契约(人工通道):回填票号即税务 ISSUED;已开具不可重复回填。
func TestPGStore_BackfillTaxNo(t *testing.T) {
	t.Run("回填成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "ISSUED", "24122000000012345678").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).BackfillTaxNo(context.Background(), 11, "24122000000012345678"); err != nil {
			t.Fatalf("backfill: %v", err)
		}
	})
	t.Run("已开具被拒", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "ISSUED", "x").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		err := NewPGStore(mock).BackfillTaxNo(context.Background(), 11, "x")
		if !errors.Is(err, ErrInvoiceNotTaxable) {
			t.Fatalf("err=%v, want ErrInvoiceNotTaxable", err)
		}
	})
}

// TestPGStore_MarkTaxResult 契约(网关回执):失败留痕/外部ID幂等/ISSUED不被乱序回退。
func TestPGStore_MarkTaxResult(t *testing.T) {
	t.Run("失败留痕可重试", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO invoice_tax_events`).
			WithArgs(int64(11), "FAILED", "", "signature invalid", "ext-001").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "FAILED", "", "signature invalid").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).MarkTaxResult(context.Background(), 11,
			TaxReceipt{Status: TaxStatusFailed, FailReason: "signature invalid", ExternalID: "ext-001"}); err != nil {
			t.Fatalf("mark failed: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("BLOCKED不伪造成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO invoice_tax_events`).
			WithArgs(int64(11), "BLOCKED", "", "external credential unavailable", "ext-002").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "BLOCKED", "", "external credential unavailable").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).MarkTaxResult(context.Background(), 11,
			TaxReceipt{Status: TaxStatusBlocked, FailReason: "external credential unavailable", ExternalID: "ext-002"}); err != nil {
			t.Fatalf("mark blocked: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("重复回执被幂等吞掉", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO invoice_tax_events`).
			WithArgs(int64(11), "FAILED", "", "timeout", "ext-003").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "FAILED", "", "timeout").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).MarkTaxResult(context.Background(), 11,
			TaxReceipt{Status: TaxStatusFailed, FailReason: "timeout", ExternalID: "ext-003"}); err != nil {
			t.Fatalf("first receipt: %v", err)
		}
		// duplicate: INSERT still happens but ON CONFLICT makes it a no-op
		mock.ExpectExec(`INSERT INTO invoice_tax_events`).
			WithArgs(int64(11), "FAILED", "", "timeout", "ext-003").
			WillReturnResult(pgxmock.NewResult("INSERT", 0))
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "FAILED", "", "timeout").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0)) // already FAILED, harmless
		if err := NewPGStore(mock).MarkTaxResult(context.Background(), 11,
			TaxReceipt{Status: TaxStatusFailed, FailReason: "timeout", ExternalID: "ext-003"}); err != nil {
			t.Fatalf("duplicate receipt: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("ISSUED不被乱序失败回退", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`INSERT INTO invoice_tax_events`).
			WithArgs(int64(11), "FAILED", "", "late failure", "ext-004").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE invoices SET tax_status`).
			WithArgs(int64(11), "FAILED", "", "late failure").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		if err := NewPGStore(mock).MarkTaxResult(context.Background(), 11,
			TaxReceipt{Status: TaxStatusFailed, FailReason: "late failure", ExternalID: "ext-004"}); err != nil {
			t.Fatalf("late failure not regressed: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("无ExternalID兼容旧调用方", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		// INSERT always called; empty external_id always inserted (no conflict)
		mock.ExpectExec(`INSERT INTO invoice_tax_events`).
			WithArgs(int64(11), "FAILED", "", "generic error", "").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE invoices SET tax_status`).WithArgs(int64(11), "FAILED", "", "generic error").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		if err := NewPGStore(mock).MarkTaxResult(context.Background(), 11,
			TaxReceipt{Status: TaxStatusFailed, FailReason: "generic error"}); err != nil {
			t.Fatalf("backward compat: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
