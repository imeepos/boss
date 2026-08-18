// 订单管理页:契约 GET /orders(keyword/status 过滤);跟踪抽屉 GET /orders/:orderNo(order+timeline)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type OrderListRow, type TimelineRow } from '../types'
import '../../org/org.css'

const STATUSES = ['PENDING', 'RESERVED', 'INSTALLING', 'DONE'] as const

export default function OrderPage() {
  const t = useT()
  const o = t.pages.orderPage
  const [rows, setRows] = useState<OrderListRow[]>([])
  const [error, setError] = useState('')
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [track, setTrack] = useState<{ order: OrderListRow; timeline: TimelineRow[] } | null>(null)
  const [trackError, setTrackError] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: OrderListRow[] }>('/orders', {
      query: { keyword: keyword || undefined, status: status || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : o.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openTrack = (orderNo: string) => {
    setTrack(null)
    setTrackError('')
    apiFetch<{ order: OrderListRow; timeline: TimelineRow[] }>(`/orders/${encodeURIComponent(orderNo)}`)
      .then((d) => setTrack({ order: d?.order ?? null as unknown as OrderListRow, timeline: d?.timeline ?? [] }))
      .catch(() => setTrackError(o.trackFail))
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={o.title} desc={o.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" placeholder={o.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <select className="org-select" value={status}
            onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{o.allStatus}</option>
            {STATUSES.map((s, i) => <option key={s} value={s}>{o.statusOptions[i]}</option>)}
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{o.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.orderNo}>
                    <td>{r.orderNo}</td>
                    <td>{r.customer || '—'}</td>
                    <td>{r.product || '—'}</td>
                    <td>{r.address || '—'}</td>
                    <td>{r.stage}. {r.stageLabel}</td>
                    <td><StatusTag domain="order" value={r.status} /></td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => openTrack(r.orderNo)}>{o.track}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{o.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(o)} />
        </div>
      </div>
      {(track || trackError) && (
      <Drawer title={o.trackTitle} onClose={() => { setTrack(null); setTrackError('') }}
        footer={<button className="org-btn org-btn-primary" onClick={() => setTrack(null)}>
          {t.pages.company.cancel}
        </button>}>
        {trackError ? <div className="org-error">{trackError}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{o.timelineColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {(track?.timeline ?? []).map((x) => (
                  <tr key={x.stage}>
                    <td>{x.stage}. {x.name}</td>
                    <td>{x.finishedAt ? fmtTime(x.finishedAt) : '—'}</td>
                    <td>{x.duration || '—'}</td>
                    <td>{x.retries}</td>
                    <td><StatusTag domain="ticket" value={x.result} /></td>
                  </tr>
                ))}
                {track !== null && !track.timeline.length && (
                  <tr><td colSpan={5}><div className="org-empty">{o.empty}</div></td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Drawer>
      )}
    </div>
  )
}
