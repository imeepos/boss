// 积分对账 Tab:按客户核对账本余额 vs 流水合计,漂移徽标 + drift 过滤。
import { useEffect, useState } from 'react'
import { pointsRecon, type PointReconRow, type PointReconSummary } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import {
  PageHead, pagerTexts, ErrorBanner, ToolbarButton, DataTable, type ColumnDef,
} from '../../../components/business'
import { Pagination } from '../../../components/Pagination'

export default function PointsReconTab() {
  const t = useT()
  const m = t.pages.marketing
  const [rows, setRows] = useState<PointReconRow[]>([])
  const [summary, setSummary] = useState<PointReconSummary | null>(null)
  const [driftOnly, setDriftOnly] = useState(false)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = () => {
    setError('')
    pointsRecon(driftOnly ? 'drift' : undefined)
      .then((d) => {
        setRows(d?.rows ?? [])
        setSummary(d?.summary ?? null)
      })
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, [driftOnly]) // eslint-disable-line react-hooks/exhaustive-deps

  const columns: ColumnDef[] = [
    { key: 'customerId', label: m.reconCustomer, render: (r) => <span className="font-medium">{String(r.customerId ?? '—')}</span> },
    { key: 'balance', label: m.reconBalance, render: (r) => String(r.balance ?? '—') },
    { key: 'entriesSum', label: m.reconEntriesSum, render: (r) => String(r.entriesSum ?? '—') },
    { key: 'lifetimeEarn', label: m.reconLifetimeEarn, render: (r) => String(r.lifetimeEarn ?? '—') },
    { key: 'expiredTotal', label: m.reconExpiredTotal, render: (r) => String(r.expiredTotal ?? '—') },
    { key: 'diff', label: m.reconDiff, render: (r) => (
      <Badge variant={r.diffKind === 'MATCH' ? 'success' : 'danger'}>
        {r.diffKind === 'MATCH' ? m.diffMatch : m.diffDrift}
      </Badge>
    ) },
  ]

  const paged = rows.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div>
      <PageHead title={m.reconTitle} desc={m.reconDesc} />
      <div className="mb-3 flex items-center gap-3">
        {summary && (
          <div className="flex-1 text-xs text-[var(--shell-crumb-text)]">
            {m.reconPointsSummaryTpl
              .replace('{n}', String(summary.customers))
              .replace('{bal}', String(summary.balanceTotal))
              .replace('{earned}', String(summary.lifetimeEarn))
              .replace('{expired}', String(summary.expiredTotal))}
          </div>
        )}
        <label className="flex cursor-pointer items-center gap-1 text-xs text-[var(--shell-content-text)]">
          <input type="checkbox" checked={driftOnly} onChange={(e) => { setDriftOnly(e.target.checked); setPage(1) }} />
          {m.reconDriftOnly}
        </label>
        <ToolbarButton onClick={load}>{m.reconRefresh}</ToolbarButton>
      </div>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <DataTable columns={columns} rows={paged.map((r) => ({ ...r }))} emptyText={m.reconAllMatch} />
        )}
        {!error && rows.length > 0 && (
          <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
            <Pagination total={rows.length} page={page} pageSize={pageSize}
              onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(m)} />
          </div>
        )}
      </Card>
    </div>
  )
}
