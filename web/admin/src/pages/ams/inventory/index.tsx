// 库存查询页:契约 GET /procurement/inventory。
// 实时聚合 assets WHERE status='IN_STOCK';迁移 000163。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { type InventoryRow } from '../types'

export default function InventoryPage() {
  const t = useT()
  const d = t.pages.inventoryPage
  const [rows, setRows] = useState<InventoryRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [materialCode, setMaterialCode] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: InventoryRow[] }>('/procurement/inventory', {
      query: { materialCode: materialCode || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [materialCode]) // eslint-disable-line react-hooks/exhaustive-deps

  const totalInStock = rows.reduce((acc, r) => acc + r.inStockQty, 0)
  const slice = rows.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div className="space-y-4">
      <PageHead title={d.title} desc={d.subtitle} />

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="rounded-lg border border-shell-divider bg-shell-bg-card p-4">
          <div className="text-xs text-shell-fg-muted">{d.metricBatches}</div>
          <div className="text-2xl font-semibold">{rows.length}</div>
        </div>
        <div className="rounded-lg border border-shell-divider bg-shell-bg-card p-4">
          <div className="text-xs text-shell-fg-muted">{d.metricInStock}</div>
          <div className="text-2xl font-semibold text-brand-primary">{totalInStock}</div>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <input
          type="text"
          value={materialCode}
          onChange={(e) => setMaterialCode(e.target.value)}
          placeholder={d.filterPlaceholder}
          className="h-9 px-3 rounded border border-shell-divider bg-shell-bg-card text-sm w-64"
        />
      </div>

      {error && <div className="text-sm text-status-danger">{error}</div>}

      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-shell-fg-muted">
            <th className="py-2 pr-4">{d.colBatch}</th>
            <th className="py-2 pr-4">{d.colBatchId}</th>
            <th className="py-2 pr-4 text-right">{d.colQty}</th>
          </tr>
        </thead>
        <tbody>
          {slice.map((r, idx) => (
            <tr key={idx} className="border-t border-shell-divider">
              <td className="py-2 pr-4 font-mono">{r.materialCode}</td>
              <td className="py-2 pr-4 font-mono">#{r.batchId}</td>
              <td className="py-2 pr-4 text-right">{r.inStockQty}</td>
            </tr>
          ))}
          {rows.length === 0 && !busy && (
            <tr><td colSpan={3} className="py-6 text-center text-shell-fg-muted">{d.empty}</td></tr>
          )}
        </tbody>
      </table>

      <Pagination
        page={page}
        pageSize={pageSize}
        total={rows.length}
        onPage={setPage}
        onSize={setPageSize}
        {...pagerTexts(t.pages.company)}
      />
    </div>
  )
}
