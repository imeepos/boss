// 缴费管理页:列名按 fields.md 裁剪(后端 Payment 无客户/时间/凭证列);契约 GET /payments(billId 过滤)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type PaymentRow } from '../types'
import { fmtFee } from '../../../lib/format'
import '../../org/org.css'

export default function PaymentPage() {
  const t = useT()
  const p = t.pages.payment
  const [rows, setRows] = useState<PaymentRow[]>([])
  const [error, setError] = useState('')
  const [billId, setBillId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: PaymentRow[] }>('/payments', {
      query: { billId: billId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" type="number" placeholder={p.filterBill}
            value={billId} onChange={(e) => { setBillId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{p.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.payNo}</td>
                    <td>#{r.billId}</td>
                    <td>{fmtFee(r.amount)}</td>
                    <td>{p.methods[r.method] ?? r.method}</td>
                    <td><StatusTag domain="payment" value={r.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{p.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
    </div>
  )
}
