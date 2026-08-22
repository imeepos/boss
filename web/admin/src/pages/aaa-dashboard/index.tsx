import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead, ToolbarButton } from '../../components/business/page-head'
import { Card, CardContent, CardHeader, CardTitle } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
type Summary = {
  accounts: number
  active: number
  suspended: number
  closed: number
  cdrs: number
  unbilled: number
  authSuccess: number
  authFailed: number
}

const emptySummary: Summary = {
  accounts: 0, active: 0, suspended: 0, closed: 0,
  cdrs: 0, unbilled: 0, authSuccess: 0, authFailed: 0,
}

export default function AaaDashboardPage() {
  const t = useT()
  const a = t.pages.aaaDashboard
  const [summary, setSummary] = useState(emptySummary)
  const [loadedAt, setLoadedAt] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setBusy(true)
    setError('')
    apiFetch<{ summary: Summary }>('/aaa/summary')
      .then((data) => {
        setSummary(data?.summary ?? emptySummary)
        setLoadedAt(new Date().toISOString())
      })
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }

  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const authTotal = summary.authSuccess + summary.authFailed
  const successRate = useMemo(
    () => (authTotal ? `${((summary.authSuccess / authTotal) * 100).toFixed(1)}%` : '—'),
    [authTotal, summary.authSuccess],
  )
  const cards = [
    [a.accounts, summary.accounts, a.accountsHint],
    [a.active, summary.active, a.activeHint],
    [a.suspended, summary.suspended, a.suspendedHint],
    [a.cdrs, summary.cdrs, a.cdrsHint],
    [a.unbilled, summary.unbilled, a.unbilledHint],
    [a.authRate, successRate, a.authRateHint],
  ] as const

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="mb-4 flex items-center gap-2">
        <ToolbarButton disabled={busy} onClick={load}>{busy ? a.refreshing : a.refresh}</ToolbarButton>
        {loadedAt && <span className="text-xs text-[var(--shell-crumb-text)]">{a.updatedAt}: {new Date(loadedAt).toLocaleString()}</span>}
      </div>
      {error && <div className="mb-4 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-xs text-[var(--color-danger)]">{error}</div>}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {cards.map(([label, value, hint]) => (
          <Card key={label} className="mb-0">
            <CardHeader><CardTitle>{label}</CardTitle></CardHeader>
            <CardContent>
              <div className="text-2xl font-semibold text-[var(--shell-heading)]">{value}</div>
              <div className="mt-1 text-xs text-[var(--shell-crumb-text)]">{hint}</div>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card className="mt-4">
        <CardHeader><CardTitle>{a.statusTitle}</CardTitle></CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <Badge variant="success">{a.success}: {summary.authSuccess}</Badge>
          <Badge variant="danger">{a.failed}: {summary.authFailed}</Badge>
          <Badge variant="warning">{a.unbilled}: {summary.unbilled}</Badge>
        </CardContent>
      </Card>
    </div>
  )
}
