package billing

import (
	"context"
	"fmt"
	"sort"
)

// ledgerDiffPriority 差异排序权重:优先展示需人工处理的差异行。
var ledgerDiffPriority = map[string]int{
	LedgerDiffRefunded: 0, LedgerDiffPartial: 1, LedgerDiffOverpaid: 2,
	LedgerDiffUnpaid: 3, LedgerDiffPaidNoInvoice: 4, LedgerDiffMatch: 5,
}

const ledgerReconSQL = `
SELECT b.id, b.bill_no, b.customer_id, b.customer_name, b.legal_entity_id, b.legal_entity_name,
       b.period, b.amount,
       COALESCE(p.paid, 0), COALESCE(p.refunded, 0),
       COALESCE(i.inv_amount, 0), COALESCE(i.invoice_no, ''), COALESCE(i.tax_status, '')
FROM bills b
LEFT JOIN LATERAL (
    SELECT SUM(amount) FILTER (WHERE status = 'SUCCESS') AS paid,
           SUM(amount) FILTER (WHERE status = 'REFUNDED') AS refunded
    FROM payments WHERE bill_id = b.id
) p ON true
LEFT JOIN LATERAL (
    SELECT total_amount AS inv_amount, invoice_no, tax_status
    FROM invoices WHERE bill_id = b.id AND status = 'ISSUED' LIMIT 1
) i ON true
LEFT JOIN regions rg ON rg.id = b.region_id
WHERE b.period = $1 AND ($2 = 0 OR b.legal_entity_id = $2)
  AND ($3 = '' OR rg.path <@ $3::ltree)`

// LedgerRecon 账实核对:账期全量账单三角比对,Go 侧分类排序后分页。
// 账期行数=月账单量(千级),SQL 分页留作后续优化;正确性与可定位优先。
func (s *PGStore) LedgerRecon(ctx context.Context, q LedgerReconQuery) ([]LedgerReconRow, int, LedgerReconSummary, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 200 {
		q.PageSize = 50
	}
	rows, err := s.db.Query(ctx, ledgerReconSQL, q.Period, q.LegalEntityID, q.RegionScope)
	if err != nil {
		return nil, 0, LedgerReconSummary{}, fmt.Errorf("billing: ledger recon query: %w", err)
	}
	defer rows.Close()
	all := make([]LedgerReconRow, 0, 128)
	for rows.Next() {
		var r LedgerReconRow
		if err := rows.Scan(&r.BillID, &r.BillNo, &r.CustomerID, &r.CustomerName, &r.LegalEntityID,
			&r.LegalEntityName, &r.Period, &r.BillAmount, &r.PaidAmount, &r.RefundAmount,
			&r.InvoiceAmount, &r.InvoiceNo, &r.TaxStatus); err != nil {
			return nil, 0, LedgerReconSummary{}, fmt.Errorf("billing: ledger recon scan: %w", err)
		}
		r.DiffKind = ClassifyLedgerRow(r)
		all = append(all, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, LedgerReconSummary{}, fmt.Errorf("billing: ledger recon rows: %w", err)
	}
	sort.SliceStable(all, func(x, y int) bool {
		px, py := ledgerDiffPriority[all[x].DiffKind], ledgerDiffPriority[all[y].DiffKind]
		if px != py {
			return px < py
		}
		return all[x].BillID < all[y].BillID
	})
	summary := summarizeLedger(all)
	start := (q.Page - 1) * q.PageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + q.PageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], len(all), summary, nil
}

// summarizeLedger 汇总三角总额与差异计数(MATCH 计数一并返回,便于口径核对)。
func summarizeLedger(rows []LedgerReconRow) LedgerReconSummary {
	s := LedgerReconSummary{ByKind: make(map[string]int, 6)}
	for _, r := range rows {
		s.BillsTotal += r.BillAmount
		s.PaidTotal += r.PaidAmount
		s.InvoiceTotal += r.InvoiceAmount
		s.ByKind[r.DiffKind]++
	}
	return s
}
