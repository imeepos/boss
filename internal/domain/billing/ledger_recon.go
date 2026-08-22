package billing

import "context"

// 账实核对差异种类(按账单定位,见 docs/design/q3-ledger-recon.md)。
const (
	LedgerDiffUnpaid        = "UNPAID"          // 未收:实收为 0
	LedgerDiffPartial       = "PARTIAL"         // 部分收:0 < 实收 < 应收
	LedgerDiffOverpaid      = "OVERPAID"        // 多收:实收 > 应收
	LedgerDiffRefunded      = "REFUNDED"        // 退款后未补收
	LedgerDiffPaidNoInvoice = "PAID_NO_INVOICE" // 已收足额未开票
	LedgerDiffMatch         = "MATCH"           // 应收=实收=开票,三角一致
)

// LedgerReconRow 账实核对行:一账单的应收/实收/开票三角与差异结论。
type LedgerReconRow struct {
	BillID          int64   `json:"billId"`
	BillNo          string  `json:"billNo"`
	CustomerID      int64   `json:"customerId"`
	CustomerName    string  `json:"customerName"`
	LegalEntityID   int64   `json:"legalEntityId"`
	LegalEntityName string  `json:"legalEntityName"`
	Period          string  `json:"period"`
	BillAmount      float64 `json:"billAmount"`
	PaidAmount      float64 `json:"paidAmount"`
	InvoiceAmount   float64 `json:"invoiceAmount"`
	RefundAmount    float64 `json:"refundAmount"`
	DiffKind        string  `json:"diffKind"`
	InvoiceNo       string  `json:"invoiceNo"`
	TaxStatus       string  `json:"taxStatus"`
}

// LedgerReconSummary 账期汇总:总应收/总实收/总开票 + 差异行计数。
type LedgerReconSummary struct {
	BillsTotal   float64        `json:"billsTotal"`
	PaidTotal    float64        `json:"paidTotal"`
	InvoiceTotal float64        `json:"invoiceTotal"`
	ByKind       map[string]int `json:"byKind"`
}

// LedgerReconQuery 账实核对查询参数。RegionScope 为账号数据范围(空=全集团)。
type LedgerReconQuery struct {
	Period        string
	LegalEntityID int64
	RegionScope   string
	Page          int
	PageSize      int
}

// ClassifyLedgerRow 纯函数:按应收/实收/开票三角判定差异种类(金额按分比较)。
func ClassifyLedgerRow(r LedgerReconRow) string {
	switch {
	case r.RefundAmount > 1e-9 && cents(r.PaidAmount) < cents(r.BillAmount):
		return LedgerDiffRefunded
	case cents(r.PaidAmount) > cents(r.BillAmount):
		return LedgerDiffOverpaid
	case cents(r.PaidAmount) == 0:
		return LedgerDiffUnpaid
	case cents(r.PaidAmount) < cents(r.BillAmount):
		return LedgerDiffPartial
	case cents(r.InvoiceAmount) == 0:
		return LedgerDiffPaidNoInvoice
	default:
		return LedgerDiffMatch
	}
}

// LedgerReconService 账实核对域服务口(账期+法人维度,逐账单定位差异)。
type LedgerReconService interface {
	LedgerRecon(ctx context.Context, q LedgerReconQuery) ([]LedgerReconRow, int, LedgerReconSummary, error)
}
