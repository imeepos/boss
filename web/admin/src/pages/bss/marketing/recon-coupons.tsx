// 券对账 Tab:按模板三角(计数器/实例/核销)展示,差异徽标 + drift 过滤。
import { useEffect, useState } from 'react'
import { couponRecon, type CouponReconRow, type CouponReconSummary } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import {
  PageHead, pagerTexts, ErrorBanner, ToolbarButton, DataTable, type ColumnDef,
} from '../../../components/business'
import { Pagination } from '../../../components/Pagination'

function DiffBadge({ kind }: { kind: CouponReconRow['diffKind'] }) {
  const m = useT().pages.marketing
  const variantMap: Record<string, 'success' | 'warning' | 'danger' | 'default'> = {
    MATCH: 'success', COUNTER_DRIFT: 'warning', REDEMPTION_LOST: 'danger',
  }
  const labelMap: Record<string, string> = {
    MATCH: m.diffMatch, COUNTER_DRIFT: m.diffCounterDrift, REDEMPTION_LOST: m.diffRedemptionLost,
  }
  return <Badge variant={variantMap[kind] ?? 'default'}>{labelMap[kind] ?? kind}</Badge>
}

export default function CouponReconTab() {
  const t = useT()
  const m = t.pages.marketing
  const [rows, setRows] = useState<CouponReconRow[]>([])
  const [summary, setSummary] = useState<CouponReconSummary | null>(null)
  const [driftOnly, setDriftOnly] = useState(false)
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const load = () => {
    setError('')
    couponRecon(driftOnly ? 'drift' : undefined)
      .then((d) => {
        setRows(d?.rows ?? [])
        setSummary(d?.summary ?? null)
      })
      .catch((e) => setError(e instanceof Error ? e.message : m.loadFail))
  }
  useEffect(load, [driftOnly]) // eslint-disable-line react-hooks/exhaustive-deps

  const columns: ColumnDef[] = [
    { key: 'name', label: m.colName, render: (r) => <span className="font-medium">{String(r.name ?? '—')}</span> },
    { key: 'issuedQty', label: m.reconIssuedQty, render: (r) => String(r.issuedQty ?? '—') },
    { key: 'actualIssued', label: m.reconActualIssued, render: (r) => String(r.actualIssued ?? '—') },
    { key: 'used', label: m.reconUsed, render: (r) => String(r.usedCount) + '/' + String(r.redemptionCnt) },
    { key: 'redeemedAmount', label: m.reconRedeemedAmount, render: (r) => (Number(r.redeemedAmount) / 100).toFixed(2) },
    { key: 'faceValueTotal', label: m.reconFaceValueTotal, render: (r) => (Number(r.faceValueTotal) / 100).toFixed(2) },
    { key: 'diff', label: m.reconDiff, render: (r) => <DiffBadge kind={r.diffKind as CouponReconRow['diffKind']} /> },
  ]

  const paged = rows.slice((page - 1) * pageSize, page * pageSize)

  return (
    <div>
      <PageHead title={m.reconTitle} desc={m.reconDesc} />
      <div className="mb-3 flex items-center gap-3">
        {summary && (
          <div className="flex-1 text-xs text-[var(--shell-crumb-text)]">
            {m.reconSummaryTpl
              .replace('{tpl}', String(summary.templates))
              .replace('{issued}', String(summary.actualIssued))
              .replace('{used}', String(summary.usedCount))
              .replace('{amount}', (summary.redeemedAmount / 100).toFixed(2))}
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
