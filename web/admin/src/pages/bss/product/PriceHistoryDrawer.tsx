// 调价记录抽屉:GET /products/:id/price-history(customer.yaml listProductPriceHistory)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import type { PriceHistoryRow } from './types'
import { fmtTime } from '../../../lib/format'

export function PriceHistoryDrawer({
  productId, productName, onClose,
}: { productId: number; productName: string; onClose: () => void }) {
  const t = useT()
  const p = t.pages.product
  const [rows, setRows] = useState<PriceHistoryRow[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<{ items: PriceHistoryRow[] }>(`/products/${productId}/price-history`)
      .then((d) => setRows(d?.items ?? []))
      .catch(() => setError(p.loadFail))
  }, [productId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={`${p.historyTitle} · ${productName}`} onClose={onClose}
      footer={<button className="org-btn org-btn-primary" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? <div className="org-error">{error}</div> : (
        <div className="org-table-wrap">
          <table className="org-table">
            <thead><tr>{p.historyColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
            <tbody>
              {(rows ?? []).map((r) => (
                <tr key={r.id}>
                  <td>{fmtFee(r.oldMonthlyFee)}</td>
                  <td>{fmtFee(r.newMonthlyFee)}</td>
                  <td>{fmtTime(r.effectiveAt)}</td>
                  <td>{r.reason || '—'}</td>
                  <td>{r.operatorAccountId ? `#${r.operatorAccountId}` : '—'}</td>
                </tr>
              ))}
              {rows !== null && !rows.length && (
                <tr><td colSpan={5}><div className="org-empty">{p.noHistory}</div></td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Drawer>
  )
}

export function fmtFee(n: number): string {
  return Number(n).toFixed(2)
}
