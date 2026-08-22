package billing

import "testing"

func TestClassifyLedgerRow(t *testing.T) {
	tests := []struct {
		name string
		row  LedgerReconRow
		want string
	}{
		{"未收", LedgerReconRow{BillAmount: 100, PaidAmount: 0, InvoiceAmount: 0}, LedgerDiffUnpaid},
		{"部分收", LedgerReconRow{BillAmount: 100, PaidAmount: 60}, LedgerDiffPartial},
		{"多收", LedgerReconRow{BillAmount: 100, PaidAmount: 120}, LedgerDiffOverpaid},
		{"退款未补", LedgerReconRow{BillAmount: 100, PaidAmount: 40, RefundAmount: 60}, LedgerDiffRefunded},
		{"已收未开票", LedgerReconRow{BillAmount: 100, PaidAmount: 100, InvoiceAmount: 0}, LedgerDiffPaidNoInvoice},
		{"三角一致", LedgerReconRow{BillAmount: 100, PaidAmount: 100, InvoiceAmount: 100}, LedgerDiffMatch},
		{"分位比较一致", LedgerReconRow{BillAmount: 100.10, PaidAmount: 100.10, InvoiceAmount: 100.10}, LedgerDiffMatch},
	}
	for _, tt := range tests {
		if got := ClassifyLedgerRow(tt.row); got != tt.want {
			t.Errorf("%s: got %s want %s", tt.name, got, tt.want)
		}
	}
}
