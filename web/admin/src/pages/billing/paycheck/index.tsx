// 渠道对账页:契约 GET /reconciliations;差异挂起批次 POST /reconciliations/:batchNo/settle。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ReconRow } from '../types'
import { fmtFee, fmtTime } from '../../../lib/format'
import '../../org/org.css'

export default function PayCheckPage() {
  const t = useT()
  const p = t.pages.paycheck
  const [rows, setRows] = useState<ReconRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReconRow[] }>('/reconciliations')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const settle = async (batchNo: string) => {
    if (busy) return
    if (!window.confirm(p.settleConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/reconciliations/${encodeURIComponent(batchNo)}/settle`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="org-card">
        <div className="org-toolbar">
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
                    <td>{r.batchNo}</td>
                    <td>{r.channel}</td>
                    <td>{fmtFee(r.channelAmount)}</td>
                    <td>{fmtFee(r.systemAmount)}</td>
                    <td>{fmtFee(r.diff)}</td>
                    <td><StatusTag domain="recon" value={r.status} /></td>
                    <td>
                      {r.status === 'DIFF_PENDING' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => settle(r.batchNo)}>{p.settle}</button>
                        </span>
                      ) : fmtTime(r.settledAt ?? '')}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{p.empty}</div></td></tr>}
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
