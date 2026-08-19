// 激活回调页(订单第 11 环节):契约 GET /activation-callbacks(裸列表)+ POST /activation-callbacks/:id/retry。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ActivationCallbackRow } from '../types'
import '../../org/org.css'

export default function CallbackPage() {
  const t = useT()
  const c = t.pages.callbackPage
  const [rows, setRows] = useState<ActivationCallbackRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<ActivationCallbackRow[]>('/activation-callbacks')
      .then((x) => setRows(Array.isArray(x) ? x : []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const retry = async (id: number) => {
    if (busy || !window.confirm(c.retryConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/activation-callbacks/${id}/retry`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{c.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>#{x.orderId}</td>
                    <td>{x.result === 'SUCCESS' ? 'SUCCESS' : x.result}</td>
                    <td>{x.retries}</td>
                    <td>
                      {x.result === 'FAILED' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => retry(x.id)}>{c.retry}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{c.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </div>
    </div>
  )
}
