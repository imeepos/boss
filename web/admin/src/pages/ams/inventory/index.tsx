// 库存查询页:契约 GET /procurement/inventory。
// 实时聚合 assets WHERE status='IN_STOCK';迁移 000163。
// 样式对齐 provision/provision:大卡片包列表 + StatCard 统计 + TableStateRow 空行。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { StatCard } from '../../../components/business/charts'
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
  const [debounced, setDebounced] = useState('')
  const [batches, setBatches] = useState<{ id: number; code: string; name: string }[]>([])

  // 防抖 300ms:与 asset/tag 检索口径一致,避免每键击发请求。
  useEffect(() => {
    const h = setTimeout(() => setDebounced(materialCode.trim()), 300)
    return () => clearTimeout(h)
  }, [materialCode])

  const load = useCallback(() => {
    setError('')
    setBusy(true)
    apiFetch<{ items: InventoryRow[] }>('/procurement/inventory', {
      query: { materialCode: debounced || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }, [debounced])
  useEffect(load, [load])

  useEffect(() => {
    apiFetch<{ items: { id: number; code: string; name: string }[] }>('/assets/batches')
      .then((d) => setBatches(d?.items ?? []))
      .catch(() => setBatches([]))
  }, [])

  const batchLabel = (id: number) => {
    const b = batches.find((x) => x.id === id)
    return b ? [b.code, b.name].filter(Boolean).join(' ') || '#' + id : '#' + id
  }

  const totalInStock = rows.reduce((acc, r) => acc + r.inStockQty, 0)
  const slice = rows.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div>
      <PageHead title={d.title} desc={d.subtitle} />

      <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <StatCard label={d.metricBatches} value={rows.length} />
        <StatCard label={d.metricInStock} value={totalInStock} />
      </div>

      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input
            type="text"
            value={materialCode}
            onChange={(e) => setMaterialCode(e.target.value)}
            placeholder={d.filterPlaceholder}
            className="h-8 w-64 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
          />
          <span className="spacer" />
          <button
            type="button"
            onClick={load}
            disabled={busy}
            className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {t.pages.audit.refresh}
          </button>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead>
                <tr className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colBatch}</th>
                  <th className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colBatchId}</th>
                  <th className="h-11 px-3 text-right text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{d.colQty}</th>
                </tr>
              </thead>
              <tbody>
                {slice.map((r, idx) => (
                  <tr key={idx} className="hover:bg-[var(--shell-menu-hover-bg)]">
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono">{r.materialCode}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono">{batchLabel(r.batchId)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-right">{r.inStockQty}</td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={3} loading={busy} text={d.empty} />}
              </tbody>
            </table>
          </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination
            total={rows.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={setPageSize}
            {...pagerTexts(t.pages.company)}
          />
        </div>
      </div>
    </div>
  )
}
