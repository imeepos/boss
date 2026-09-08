// 实名核验记录抽屉:GET /customers/:id/verify-logs(customer.yaml listCustomerVerifyLogs)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import { Table, TableBody, TableHead, TableHeader, TableRow, TableCell } from '../../../components/ui/table'
import { ErrorBanner } from '../../../components/business'
import type { VerifyLogRow } from './types'
import { fmtTime } from '../../../lib/format'
import { EmptyState } from '../../../components/business'

export function VerifyLogsDrawer({
  customerId, customerName, onClose,
}: { customerId: number; customerName: string; onClose: () => void }) {
  const t = useT()
  const c = t.pages.customer
  const [rows, setRows] = useState<VerifyLogRow[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<{ items: VerifyLogRow[] }>(`/customers/${customerId}/verify-logs`)
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : c.verifyFail))
  }, [customerId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={`${c.verifyTitle} · ${customerName}`} onClose={onClose}
      footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? <ErrorBanner message={error} /> : (
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>#</TableHead>
                {c.verifyColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}
              </TableRow>
            </TableHeader>
            <TableBody>
              {(rows ?? []).map((r, i) => (
                <TableRow key={r.id}>
                  <TableCell>{i + 1}</TableCell>
                  <TableCell>{r.method}</TableCell>
                  <TableCell>{fmtTime(r.verifiedAt)}</TableCell>
                  <TableCell><VerifyResultBadge result={r.result} labels={c.verifyResultLabels} /></TableCell>
                  <TableCell>{r.operatorName || `#${r.operatorAccountId}` || '—'}</TableCell>
                </TableRow>
              ))}
              {rows !== null && !rows.length && (
                <TableRow><TableCell colSpan={5}><EmptyState text={c.empty} /></TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      )}
    </Drawer>
  )
}

/** 核验结论三态徽标:PENDING(自助提交在途)不得渲染成"不通过"。 */
function VerifyResultBadge({ result, labels }: { result: string; labels: Record<string, string> }) {
  const color = result === 'PASS' ? 'var(--color-success)' : result === 'FAIL' ? 'var(--color-danger)' : 'var(--color-warning)'
  const label = labels[result] ?? result
  return (
    <span className="inline-flex h-6 items-center rounded-full px-2 text-[11px] font-medium"
      style={{ color, background: `color-mix(in srgb, ${color} 15%, transparent)` }}>
      {label}
    </span>
  )
}
