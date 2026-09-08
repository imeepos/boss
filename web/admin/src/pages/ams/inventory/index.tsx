// 库存查询页:契约 GET /procurement/inventory。
// 实时聚合 assets WHERE status='IN_STOCK';迁移 000163。
// 样式对齐 provision/provision:大卡片包列表 + StatCard 统计 + TableStateRow 空行。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { TableStateRow, ToolbarButton } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { StatCard } from '../../../components/business/charts'
import { Card, CardContent, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
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

      <Card>
        <CardContent>
          <div className="flex flex-wrap items-center gap-2">
            <input
              type="text"
              value={materialCode}
              onChange={(e) => setMaterialCode(e.target.value)}
              placeholder={d.filterPlaceholder}
              className="h-8 w-64 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
            />
            <span className="spacer" />
            <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
          </div>
        </CardContent>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{d.colBatch}</TableHead>
                <TableHead>{d.colBatchId}</TableHead>
                <TableHead className="text-right">{d.colQty}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r, idx) => (
                <TableRow key={idx}>
                  <TableCell className="font-mono">{r.materialCode}</TableCell>
                  <TableCell className="font-mono">{batchLabel(r.batchId)}</TableCell>
                  <TableCell className="text-right">{r.inStockQty}</TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={3} loading={busy} text={d.empty} />}
            </TableBody>
          </Table>
        </div>
        <CardFooter>
          <Pagination
            total={rows.length}
            page={page}
            pageSize={pageSize}
            onPage={setPage}
            onSize={setPageSize}
            {...pagerTexts(t.pages.company)}
          />
        </CardFooter>
      </Card>
    </div>
  )
}
