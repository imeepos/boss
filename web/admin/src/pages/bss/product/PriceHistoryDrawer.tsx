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
      footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{p.historyColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
            <tbody>
              {(rows ?? []).map((r) => (
                <tr key={r.id}>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.oldMonthlyFee)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.newMonthlyFee)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.effectiveAt)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.reason || '—'}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.operatorAccountId ? `#${r.operatorAccountId}` : '—'}</td>
                </tr>
              ))}
              {rows !== null && !rows.length && (
                <tr><td colSpan={5} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{p.noHistory}</div></td></tr>
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
