// 实名核验记录抽屉:GET /customers/:id/verify-logs(customer.yaml listCustomerVerifyLogs)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import type { VerifyLogRow } from './types'
import { fmtTime } from '../../../lib/format'

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
      .catch(() => setError(c.verifyFail))
  }, [customerId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Drawer title={`${c.verifyTitle} · ${customerName}`} onClose={onClose}
      footer={<button className="org-btn org-btn-primary" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? <div className="org-error">{error}</div> : (
        <div className="org-table-wrap">
          <table className="org-table">
            <thead><tr><th>#</th>{c.verifyColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
            <tbody>
              {(rows ?? []).map((r, i) => (
                <tr key={r.id}>
                  <td>{i + 1}</td>
                  <td>{r.method}</td>
                  <td>{fmtTime(r.verifiedAt)}</td>
                  <td>{r.result === 'PASS' ? '通过' : '不通过'}</td>
                  <td>{r.operatorName || `#${r.operatorAccountId}` || '—'}</td>
                </tr>
              ))}
              {rows !== null && !rows.length && (
                <tr><td colSpan={5}><div className="org-empty">{c.empty}</div></td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Drawer>
  )
}
