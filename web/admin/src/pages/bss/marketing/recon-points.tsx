// 积分对账 Tab:按客户核对账本余额 vs 流水合计,漂移徽标 + drift 过滤。
import { useEffect, useState } from 'react'
import { pointsRecon, type PointReconRow, type PointReconSummary } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business'

export default function PointsReconTab() {
  const t = useT()
  const m = t.pages.marketing
  const [rows, setRows] = useState<PointReconRow[]>([])
  const [summary, setSummary] = useState<PointReconSummary | null>(null)
  const [driftOnly, setDriftOnly] = useState(false)
  const [error, setError] = useState('')

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

  return (
    <div>
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
          <input type="checkbox" checked={driftOnly} onChange={(e) => setDriftOnly(e.target.checked)} />
          {m.reconDriftOnly}
        </label>
        <ToolbarButton onClick={load}>{m.reconRefresh}</ToolbarButton>
      </div>
      <Card className="p-4">
        {error ? <ErrorBanner message={error} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{m.reconCustomer}</TableHead>
                <TableHead>{m.reconBalance}</TableHead>
                <TableHead>{m.reconEntriesSum}</TableHead>
                <TableHead>{m.reconLifetimeEarn}</TableHead>
                <TableHead>{m.reconExpiredTotal}</TableHead>
                <TableHead>{m.reconDiff}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={r.customerId}>
                  <TableCell className="font-medium">{r.customerId}</TableCell>
                  <TableCell>{r.balance}</TableCell>
                  <TableCell>{r.entriesSum}</TableCell>
                  <TableCell>{r.lifetimeEarn}</TableCell>
                  <TableCell>{r.expiredTotal}</TableCell>
                  <TableCell>
                    <Badge variant={r.diffKind === 'MATCH' ? 'success' : 'danger'}>{r.diffKind}</Badge>
                  </TableCell>
                </TableRow>
              ))}
              {!rows.length && (
                <TableRow><TableCell colSpan={6}><EmptyState text={m.reconAllMatch} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  )
}
