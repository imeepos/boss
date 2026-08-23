// 券对账 Tab:按模板三角(计数器/实例/核销)展示,差异徽标 + drift 过滤。
import { useEffect, useState } from 'react'
import { couponRecon, type CouponReconRow, type CouponReconSummary } from '../../../api/marketing'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { ErrorBanner, EmptyState, ToolbarButton } from '../../../components/business'

function DiffBadge({ kind }: { kind: CouponReconRow['diffKind'] }) {
  const map: Record<string, 'success' | 'warning' | 'danger' | 'default'> = {
    MATCH: 'success', COUNTER_DRIFT: 'warning', REDEMPTION_LOST: 'danger',
  }
  return <Badge variant={map[kind] ?? 'default'}>{kind}</Badge>
}

export default function CouponReconTab() {
  const t = useT()
  const m = t.pages.marketing
  const [rows, setRows] = useState<CouponReconRow[]>([])
  const [summary, setSummary] = useState<CouponReconSummary | null>(null)
  const [driftOnly, setDriftOnly] = useState(false)
  const [error, setError] = useState('')

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

  return (
    <div>
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
                <TableHead>{m.colName}</TableHead>
                <TableHead>{m.reconIssuedQty}</TableHead>
                <TableHead>{m.reconActualIssued}</TableHead>
                <TableHead>{m.reconUsed}</TableHead>
                <TableHead>{m.reconRedeemedAmount}</TableHead>
                <TableHead>{m.reconFaceValueTotal}</TableHead>
                <TableHead>{m.reconDiff}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={r.templateId}>
                  <TableCell className="font-medium">{r.name}</TableCell>
                  <TableCell>{r.issuedQty}</TableCell>
                  <TableCell>{r.actualIssued}</TableCell>
                  <TableCell>{r.usedCount}/{r.redemptionCnt}</TableCell>
                  <TableCell>{(r.redeemedAmount / 100).toFixed(2)}</TableCell>
                  <TableCell>{(r.faceValueTotal / 100).toFixed(2)}</TableCell>
                  <TableCell><DiffBadge kind={r.diffKind} /></TableCell>
                </TableRow>
              ))}
              {!rows.length && (
                <TableRow><TableCell colSpan={7}><EmptyState text={m.reconAllMatch} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  )
}
