// 下发日志页:契约 GET /provision-logs?taskId(只读,成功/失败/重试留痕)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ProvisionLogRow } from '../types'
import '../../org/org.css'

export default function ProvisionLogPage() {
  const t = useT()
  const p = t.pages.provlogPage
  const [rows, setRows] = useState<ProvisionLogRow[]>([])
  const [error, setError] = useState('')
  const [taskId, setTaskId] = useState('')
  const [result, setResult] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ProvisionLogRow[] }>('/provision-logs', {
      query: { taskId: taskId ? Number(taskId) || undefined : undefined },
    })
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = result ? rows.filter((x) => x.result === result) : rows
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" style={{ width: 160 }} placeholder={p.filterTask}
            value={taskId} onChange={(e) => { setTaskId(e.target.value); setPage(1) }} />
          <select className="org-select" value={result}
            onChange={(e) => { setResult(e.target.value); setPage(1) }}>
            <option value="">{p.allResult}</option>
            <option value="SUCCESS">SUCCESS</option>
            <option value="FAILED">FAILED</option>
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{p.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>#{x.taskId}</td>
                    <td>{x.resourceCode || `#${x.resourceId}`}</td>
                    <td>{x.templateCode || `#${x.templateId}`}</td>
                    <td>{x.result}</td>
                    <td>{x.retries}</td>
                    <td>{fmtTime(x.createdAt)}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{p.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
    </div>
  )
}
