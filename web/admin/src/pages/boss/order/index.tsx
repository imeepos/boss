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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={o.searchPlaceholder}
            value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1) }} />
          <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={status}
            onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{o.allStatus}</option>
            {STATUSES.map((s, i) => <option key={s} value={s}>{o.statusOptions[i]}</option>)}
          </select>
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{o.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.orderNo}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.orderNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.customer || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.product || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.address || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.stage}. {r.stageLabel}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="order" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => openTrack(r.orderNo)}>{o.track}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{o.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(o)} />
        </div>
      </div>
      {(track || trackError) && (
      <Drawer title={o.trackTitle} onClose={() => { setTrack(null); setTrackError('') }}
        footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setTrack(null)}>
          {t.pages.company.cancel}
        </button>}>
        {trackError ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{trackError}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{o.timelineColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(track?.timeline ?? []).map((x) => (
                  <tr key={x.stage}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.stage}. {x.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.finishedAt ? fmtTime(x.finishedAt) : '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.duration || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.retries}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="ticket" value={x.result} /></td>
                  </tr>
                ))}
                {track !== null && !track.timeline.length && (
                  <tr><td colSpan={5} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{o.empty}</div></td></tr>
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
