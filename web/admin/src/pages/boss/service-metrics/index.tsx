// 客户服务指标页(boss 域):CS 工单 SLA + AR 应收账龄聚合指标。
import { useEffect, useState } from 'react'
import { useT } from '../../../i18n'
import { apiFetch } from '../../../api/client'
import { PageHead } from '../../org/shared'
import { CardShell } from '../../../components/business/charts'
import type { CSMetrics } from '../types'
import type { ARMetrics } from '../../billing/types'

export default function ServiceMetricsPage() {
  const t = useT()
  const c = t.pages.serviceMetricsPage
  const [cs, setCS] = useState<CSMetrics | null>(null)
  const [ar, setAR] = useState<ARMetrics | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([apiFetch<CSMetrics>('/complaint-metrics'), apiFetch<ARMetrics>('/ar-metrics')])
      .then(([m, a]) => { setCS(m); setAR(a) })
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [c.loadFail])

  const csCards = cs && [
    [c.csCards.open, cs.openCount],
    [c.csCards.processing, cs.processingCount],
    [c.csCards.slaBreached, cs.slaBreachedOpen],
    [c.csCards.avgCloseHours, cs.avgCloseHours.toFixed(1)],
  ] as const

  const arBuckets = ar && Object.entries(ar.agingBuckets)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <div className="mb-4 flex justify-end">
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      {error && (
        <div className="mb-4 text-sm text-[var(--color-danger)]">{error}</div>
      )}
      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        {csCards?.map(([label, value]) => (
          <div key={label} className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4">
            <div className="text-xs text-[var(--shell-group-title)]">{label}</div>
            <div className="mt-2 text-2xl font-semibold text-[var(--shell-heading)]">{value}</div>
          </div>
        ))}
      </div>
      <div className="grid gap-4 xl:grid-cols-2">
        <CardShell title={c.arAgingTitle}>
          <div className="grid grid-cols-2 gap-3">
            {arBuckets?.map(([key, value]) => (
              <div key={key} className="rounded-sm bg-[var(--shell-menu-hover-bg)] p-3 text-sm">
                <span>{c.agingBuckets[key] ?? key}</span>
                <strong className="float-right text-[var(--shell-heading)]">{value}</strong>
              </div>
            ))}
          </div>
        </CardShell>
        <CardShell title={c.arSummaryTitle}>
          <dl className="grid grid-cols-2 gap-4 text-sm text-[var(--shell-content-text)]">
            <dt>{c.arSummary.totalAmount}</dt>
            <dd className="text-[var(--shell-heading)]">{ar?.totalAmount ?? '—'}</dd>
            <dt>{c.arSummary.customers}</dt>
            <dd className="text-[var(--shell-heading)]">{ar?.customerCount ?? '—'}</dd>
            <dt>{c.arSummary.stopped}</dt>
            <dd className="text-[var(--shell-heading)]">{ar?.stoppedCount ?? '—'}</dd>
            <dt>{c.arSummary.overdueBills}</dt>
            <dd className="text-[var(--shell-heading)]">{ar?.overdueBillCount ?? '—'}</dd>
          </dl>
        </CardShell>
      </div>
    </div>
  )
}
