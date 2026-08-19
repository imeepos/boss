// 报告中心页:契约 GET /reports(items)+ GET /reports/latest?period(正文抽屉)+ POST /reports/:id/send。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ReportPayload, type ReportRow } from '../types'
import '../../org/org.css'

const PERIODS = ['daily', 'weekly', 'monthly', 'quarterly'] as const

export default function ReportPage() {
  const t = useT()
  const r = t.pages.reportPage
  const [rows, setRows] = useState<ReportRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [view, setView] = useState<ReportPayload | null>(null)
  const [viewError, setViewError] = useState('')
  const [notice, setNotice] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReportRow[] }>('/reports')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const viewLatest = (period: string) => {
    setView(null)
    setViewError('')
    apiFetch<{ payload: ReportPayload }>('/reports/latest', { query: { period } })
      .then((d) => setView(d?.payload ?? null))
      .catch(() => setViewError(r.viewFail))
  }

  const send = (row: ReportRow) => {
    if (busy || !window.confirm(r.sendConfirm.replace('{id}', String(row.id)))) return
    setNotice('')
    setBusy(true)
    apiFetch(`/reports/${row.id}/send`, { method: 'POST' })
      .then(() => setNotice(r.sent.replace('{id}', String(row.id))))
      .catch(() => setNotice(r.sendFail))
      .finally(() => setBusy(false))
  }

  const slice = pageSlice(rows, page, pageSize)
  const periodLabel = (p: string) => {
    const i = PERIODS.indexOf(p as typeof PERIODS[number])
    return i >= 0 ? r.periods[i] : p
  }

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          {PERIODS.map((p, i) => (
            <button key={p} className="org-btn" disabled={busy} onClick={() => viewLatest(p)}>
              {r.view} · {r.periods[i]}
            </button>
          ))}
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          {notice && <span style={{ fontSize: 13, color: '#52c41a' }}>{notice}</span>}
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{r.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>{periodLabel(x.period)}</td>
                    <td>{fmtTime(x.windowStart)}</td>
                    <td>{fmtTime(x.windowEnd)}</td>
                    <td>{fmtTime(x.createdAt)}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => viewLatest(x.period)}>{r.view}</button>
                        <button disabled={busy} onClick={() => send(x)}>{r.send}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{r.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
      {(view || viewError) && (
        <Drawer title={r.viewTitle} onClose={() => { setView(null); setViewError('') }}
          footer={<button className="org-btn org-btn-primary" onClick={() => { setView(null); setViewError('') }}>
            {t.pages.company.cancel}
          </button>}>
          {viewError ? <div className="org-error">{viewError}</div> : !view ? (
            <div className="org-empty">{r.viewEmpty}</div>
          ) : (
            <div className="org-detail-list">
              <div className="org-detail-item"><span className="k">{r.generatedAtLabel}</span>
                <span className="v">{fmtTime(view.generatedAt)}</span></div>
              {view.indicators.map((x) => (
                <div className="org-detail-item" key={x.key}>
                  <span className="k">{x.name || x.key}</span><span className="v">{x.value} · {x.detail}</span>
                </div>
              ))}
              {view.conclusions && view.conclusions.length > 0 && (
                <div className="org-detail-item"><span className="k">{r.conclusionsLabel}</span>
                  <span className="v">{view.conclusions.map((c, i) => <div key={i}>{c}</div>)}</span></div>
              )}
            </div>
          )}
        </Drawer>
      )}
    </div>
  )
}
