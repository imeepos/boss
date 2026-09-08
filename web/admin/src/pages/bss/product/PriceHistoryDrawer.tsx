// 调价记录抽屉:GET /products/:id/price-history(customer.yaml listProductPriceHistory)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '../../../components/ui/table'
import { EmptyState, ErrorBanner } from '../../../components/business'
import type { PriceHistoryRow } from './types'
import { fmtFee, fmtTime } from '../../../lib/format'

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
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
  }, [productId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={`${p.historyTitle} · ${productName}`} onClose={onClose}
      footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? <ErrorBanner message={error} /> : (
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{p.historyColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {(rows ?? []).map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{fmtFee(r.oldMonthlyFee)}</TableCell>
                  <TableCell>{fmtFee(r.newMonthlyFee)}</TableCell>
                  <TableCell>{fmtTime(r.effectiveAt)}</TableCell>
                  <TableCell>{r.reason || '—'}</TableCell>
                  <TableCell>{r.operatorAccountId ? `#${r.operatorAccountId}` : '—'}</TableCell>
                </TableRow>
              ))}
              {rows !== null && !rows.length && (
                <TableRow><TableCell colSpan={5}><EmptyState text={p.noHistory} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      )}
    </Drawer>
  )
}
