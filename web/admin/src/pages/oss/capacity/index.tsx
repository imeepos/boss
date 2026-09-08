// 容量视图页:列名以 fields.md §4.2.2 为准;契约 GET /resources/capacity + POST /resources/capacity/alert-scan。
// 使用率 >=80% 行预警高亮(与后端告警阈值 resource.CapacityWarnThresholdPct 对齐)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { pageSlice, type CapacityRow } from '../types'
import { TableStateRow, ErrorBanner, ToolbarButton } from '../../../components/business'

const ALERT_THRESHOLD_PCT = 80 // fields.md §4.2.2 预警阈值

export default function CapacityPage() {
  const t = useT()
  const r = t.pages.capacityPage
  const [rows, setRows] = useState<CapacityRow[]>([])
  const [dim, setDim] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = (d: string) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: CapacityRow[] }>('/resources/capacity', { query: { dim: d || undefined, order: 'usageDesc' } })
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => {
        const msg = e instanceof Error ? e.message : r.loadFail
        setError(msg)
        toast.error(r.loadFail, { description: msg })
      })
      .finally(() => setBusy(false))
  }

  const runScan = () => {
    setBusy(true)
    apiFetch<{ scanned: number; created: number; resolved: number }>('/resources/capacity/alert-scan', { method: 'POST' })
      .then((x) => {
        const msg = r.scanDone.replace(
          '{scanned}', String(x?.scanned ?? 0),
        ).replace('{created}', String(x?.created ?? 0)).replace('{resolved}', String(x?.resolved ?? 0))
        toast.success(msg)
        load(dim)
      })
      .catch((e) => {
        const msg = e instanceof Error ? e.message : r.scanFail
        toast.error(r.scanFail, { description: msg })
      })
      .finally(() => setBusy(false))
  }

  useEffect(() => { load('') }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const dimLabel = (d: string) => (d === 'OLT' ? r.dimOlt : r.dimSplitter)
  const dimOptions = [
    { value: '', label: r.dimAll },
    { value: 'OLT', label: r.dimOlt },
    { value: 'SPLITTER', label: r.dimSplitter },
  ]
  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Dropdown
            value={dim}
            options={dimOptions}
            onChange={(v) => { setDim(v); setPage(1); load(v) }}
            ariaLabel={r.dimAll}
            triggerStyle={{ minWidth: 160 }}
          />
          <span className="flex-1" />
          <ToolbarButton onClick={() => load(dim)} disabled={busy}>
            {t.pages.audit.refresh}
          </ToolbarButton>
          <ToolbarButton primary onClick={runScan} disabled={busy}>
            {r.scan}
          </ToolbarButton>
        </div>
        {error ? <div className="px-4 pb-3"><ErrorBanner message={error} /></div> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {[r.colObject, r.colType, r.colTotal, r.colUsed, r.colUsage].map((x) => <TableHead key={x}>{x}</TableHead>)}
                </TableRow>
              </TableHeader>
              <TableBody>
                {slice.map((row) => (
                  <TableRow key={row.resourceId} className={row.usageRate >= ALERT_THRESHOLD_PCT ? "bg-[color-mix(in_srgb,var(--color-warning)_12%,transparent)]" : undefined}>
                    <TableCell>{row.name} ({row.code})</TableCell>
                    <TableCell>{dimLabel(row.type)}</TableCell>
                    <TableCell>{row.totalPorts}</TableCell>
                    <TableCell>{row.usedPorts}</TableCell>
                    <TableCell>
                      <span className="inline-flex items-center gap-2">
                        <span>{row.usageRate.toFixed(2)}%</span>
                        {row.usageRate >= ALERT_THRESHOLD_PCT && (
                          <span title={r.alertHint} className="rounded-sm bg-[color-mix(in_srgb,var(--color-warning)_16%,transparent)] px-1.5 py-0.5 text-[11px] font-medium text-[var(--color-warning)]">{r.alertMark}</span>
                        )}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={5} loading={busy} text={r.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </CardFooter>
      </Card>
    </div>
  )
}
