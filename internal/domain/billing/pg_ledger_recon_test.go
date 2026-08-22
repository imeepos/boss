package billing

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_LedgerRecon 契约:账期三角查询 + 分类 + 差异优先排序 + 汇总。
func TestPGStore_LedgerRecon(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery(`FROM bills b`).
		WithArgs("2026-08", int64(0), "").
		WillReturnRows(mock.NewRows([]string{
			"id", "bill_no", "customer_id", "customer_name", "legal_entity_id", "legal_entity_name",
			"period", "amount", "paid", "refunded", "inv_amount", "invoice_no", "tax_status",
		}).
			AddRow(int64(2), "BILL-2", int64(9), "客户B", int64(1), "LEG", "2026-08",
				100.0, 0.0, 0.0, 0.0, "", "").
			AddRow(int64(1), "BILL-1", int64(8), "客户A", int64(1), "LEG", "2026-08",
				200.0, 200.0, 0.0, 200.0, "INV-1", "ISSUED").
			AddRow(int64(3), "BILL-3", int64(10), "客户C", int64(1), "LEG", "2026-08",
				50.0, 50.0, 0.0, 0.0, "", ""))

	items, total, summary, err := NewPGStore(mock).LedgerRecon(context.Background(),
		LedgerReconQuery{Period: "2026-08", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("LedgerRecon: %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("total=%d items=%d", total, len(items))
	}
	// 差异优先:UNPAID(BILL-2) > PAID_NO_INVOICE(BILL-3) > MATCH(BILL-1)
	if items[0].BillNo != "BILL-2" || items[0].DiffKind != LedgerDiffUnpaid {
		t.Fatalf("first=%+v", items[0])
	}
	if items[1].DiffKind != LedgerDiffPaidNoInvoice || items[2].DiffKind != LedgerDiffMatch {
		t.Fatalf("order=%s,%s", items[1].DiffKind, items[2].DiffKind)
	}
	if summary.BillsTotal != 350 || summary.PaidTotal != 250 || summary.InvoiceTotal != 200 {
		t.Fatalf("summary=%+v", summary)
	}
	if summary.ByKind[LedgerDiffUnpaid] != 1 || summary.ByKind[LedgerDiffPaidNoInvoice] != 1 || summary.ByKind[LedgerDiffMatch] != 1 {
		t.Fatalf("byKind=%+v", summary.ByKind)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_LedgerReconPagination 契约:Go 侧分页切片。
func TestPGStore_LedgerReconPagination(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	rows := mock.NewRows([]string{
		"id", "bill_no", "customer_id", "customer_name", "legal_entity_id", "legal_entity_name",
		"period", "amount", "paid", "refunded", "inv_amount", "invoice_no", "tax_status",
	})
	for i := 1; i <= 5; i++ {
		rows.AddRow(int64(i), "BILL-i", int64(i), "c", int64(1), "LEG", "2026-08",
			10.0, 10.0, 0.0, 10.0, "INV", "ISSUED")
	}
	mock.ExpectQuery(`FROM bills b`).WithArgs("2026-08", int64(0), "").WillReturnRows(rows)
	items, total, _, err := NewPGStore(mock).LedgerRecon(context.Background(),
		LedgerReconQuery{Period: "2026-08", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || len(items) != 2 {
		t.Fatalf("total=%d len=%d", total, len(items))
	}
}
