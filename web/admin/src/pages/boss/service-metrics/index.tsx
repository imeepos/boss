import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead } from '../../org/shared'
import { CardShell } from '../../../components/business/charts'
import type { CSMetrics } from '../types'
import type { ARMetrics } from '../../billing/types'

export default function ServiceMetricsPage() {
  const t = useT()
  const [cs, setCS] = useState<CSMetrics | null>(null)
  const [ar, setAR] = useState<ARMetrics | null>(null)
  const [error, setError] = useState('')
  useEffect(() => {
    Promise.all([apiFetch<CSMetrics>('/complaint-metrics'), apiFetch<ARMetrics>('/ar-metrics')])
      .then(([c, a]) => { setCS(c); setAR(a) })
      .catch((e) => setError(e instanceof Error ? e.message : 'Failed to load'))
  }, [])
  const cards = cs && [
    ['CS OPEN', cs.openCount], ['CS PROCESSING', cs.processingCount], ['SLA BREACHED', cs.slaBreachedOpen], ['AVG CLOSE HOURS', cs.avgCloseHours.toFixed(1)],
  ]
  const buckets = ar && Object.entries(ar.agingBuckets)
  return <div><PageHead title={t.menu.items['service-metrics'] ?? 'Customer service metrics'} desc="CS SLA and AR aging metrics from real backend aggregates" />
    {error && <div className="mb-4 text-sm text-[var(--color-danger)]">{error}</div>}
    <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">{cards?.map(([label, value]) => <div key={String(label)} className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4"><div className="text-xs text-[var(--shell-group-title)]">{label}</div><div className="mt-2 text-2xl font-semibold text-[var(--shell-heading)]">{value}</div></div>)}</div>
    <div className="grid gap-4 xl:grid-cols-2"><CardShell title="AR aging buckets"><div className="grid grid-cols-2 gap-3">{buckets?.map(([key, value]) => <div key={key} className="rounded-sm bg-[var(--shell-menu-hover-bg)] p-3 text-sm"><span>{key}</span><strong className="float-right">{value}</strong></div>)}</div></CardShell><CardShell title="AR summary"><dl className="grid grid-cols-2 gap-4 text-sm"><dt>Total amount</dt><dd>{ar?.totalAmount ?? '—'}</dd><dt>Customers</dt><dd>{ar?.customerCount ?? '—'}</dd><dt>Stopped</dt><dd>{ar?.stoppedCount ?? '—'}</dd><dt>Overdue bills</dt><dd>{ar?.overdueBillCount ?? '—'}</dd></dl></CardShell></div>
  </div>
}
