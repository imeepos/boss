// 订单管理页:契约 GET /orders(keyword/status 过滤);跟踪抽屉 GET /orders/:orderNo(order+timeline);
// 环节推进 POST /orders/:orderNo/{check-resource,reserve,charge,cancel}(order_workflow.go)。
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type CheckDetail, type OrderListRow, type TimelineRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, EmptyState } from '../../../components/business'

const STATUSES = ['PENDING', 'RESERVED', 'INSTALLING', 'DONE', 'CANCELLED'] as const

export default function OrderPage() {
  const t = useT()
  const nav = useNavigate()
  const confirmDialog = useConfirm()
  const o = t.pages.orderPage
  const [rows, setRows] = useState<OrderListRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [urlStatus, setUrlStatus] = useQueryState('status', '')
  const [urlPage, setUrlPage] = useQueryInt('page', 1)
  const [urlPageSize, setUrlPageSize] = useQueryInt('size', 10)
  const [keyword, setKeyword] = useState(urlKeyword)
  const [status, setStatus] = useState(urlStatus)
  const [page, setPage] = useState(urlPage)
  const [pageSize, setPageSize] = useState(urlPageSize)
  const [busy, setBusy] = useState(false)
  const [track, setTrack] = useState<{ order: OrderListRow; timeline: TimelineRow[] } | null>(null)
  const [trackError, setTrackError] = useState('')
  const [check, setCheck] = useState<{ row: OrderListRow; detail: CheckDetail | null; result: boolean | null; message: string } | null>(null)

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

  const openCheck = (row: OrderListRow) => {
    setCheck({ row, detail: null, result: null, message: '' })
    apiFetch<{ detail: CheckDetail }>(`/orders/${encodeURIComponent(row.orderNo)}/check-preview`)
      .then((d) => setCheck((x) => x ? { ...x, detail: d?.detail ?? null } : x))
      .catch((e) => setCheck((x) => x ? { ...x, message: e instanceof Error ? e.message : o.actionFail } : x))
  }

  const runCheck = () => {
    if (!check || busy) return
    setBusy(true)
    apiFetch<{ available: boolean; idlePorts: string[] }>(`/orders/${encodeURIComponent(check.row.orderNo)}/check-resource`, { method: 'POST' })
      .then((d) => setCheck((x) => x ? { ...x, result: d?.available ?? false, message: d?.available ? o.checkPass.replace('{count}', String(d.idlePorts?.length ?? 0)) : o.checkFail } : x))
      .catch((e) => setCheck((x) => x ? { ...x, message: e instanceof Error ? e.message : o.actionFail } : x))
      .finally(() => { setBusy(false); load() })
  }

  // advance 按当前环节推进:2→预占 3→收费(自动段 5-8);cancel 需二次确认。
  const reserveFromCheck = () => {
    if (!check || busy) return
    setBusy(true)
    apiFetch(`/orders/${encodeURIComponent(check.row.orderNo)}/reserve`, { method: 'POST' })
      .then(() => { setCheck(null); load() })
      .catch((e) => setCheck((x) => x ? { ...x, message: e instanceof Error ? e.message : o.actionFail } : x))
      .finally(() => setBusy(false))
  }

  const advance = async (row: OrderListRow, action: 'reserve' | 'charge' | 'cancel') => {
    if (busy) return
    if (action === 'cancel' && !(await confirmDialog(o.confirmCancel.replace('{no}', row.orderNo), { danger: true }))) return
    setBusy(true)
    setError('')
    apiFetch(`/orders/${encodeURIComponent(row.orderNo)}/${action}`, { method: 'POST' })
      .then(() => load())
      .catch((e) => setError(e instanceof Error ? e.message : o.actionFail))
      .finally(() => setBusy(false))
  }

  const statusOptions = [{ value: '', label: o.allStatus }, ...STATUSES.map((value, i) => ({ value, label: o.statusOptions[i] }))]

  const updateKeyword = (value: string) => {
    setKeyword(value)
    setUrlKeyword(value)
    setPage(1)
    setUrlPage(1)
  }

  const updateStatus = (value: string) => {
    setStatus(value)
    setUrlStatus(value)
    setPage(1)
    setUrlPage(1)
  }

  const updatePageSize = (value: number) => {
    setPageSize(value)
    setUrlPageSize(value)
    setPage(1)
    setUrlPage(1)
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={o.title} desc={o.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" placeholder={o.searchPlaceholder}
            value={keyword} onChange={(e) => updateKeyword(e.target.value)} />
          <Dropdown value={status} options={statusOptions} onChange={updateStatus} ariaLabel={o.allStatus} />
          <span className="spacer" />
          <button type="button" className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
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
                      <span className="inline-flex items-center gap-3">
                        {r.stage === 1 && r.status === 'PENDING' && (
                          <button type="button" disabled={busy} onClick={() => openCheck(r)}>{o.actCheck}</button>
                        )}
                        {r.stage === 2 && r.status === 'PENDING' && (
                          <button type="button" disabled={busy} onClick={() => openCheck(r)}>{o.actReserve}</button>
                        )}
                        {r.stage === 3 && r.status === 'RESERVED' && (
                          <button type="button" disabled={busy} onClick={() => advance(r, 'charge')}>{o.actCharge}</button>
                        )}
                        {r.status !== 'DONE' && r.status !== 'CANCELLED' && (
                          <button type="button" disabled={busy} onClick={() => advance(r, 'cancel')}>{o.actCancel}</button>
                        )}
                        <button type="button" onClick={() => openTrack(r.orderNo)}>{o.track}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={o.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={(value) => { setPage(value); setUrlPage(value) }}
            onSize={updatePageSize} {...pagerTexts(o)} />
        </div>
      </div>
      {check && (
        <Drawer title={o.checkTitle} onClose={() => setCheck(null)}
          footer={<div className="flex gap-2">
            {check.result === false && <><button type="button" className="h-8 rounded-sm border border-[var(--shell-input-border)] px-4 text-[13px]" onClick={() => nav('/oss/expand')}>{o.expand}</button><button type="button" className="h-8 rounded-sm border border-[var(--shell-input-border)] px-4 text-[13px]" onClick={() => nav('/oss/transfer')}>{o.transfer}</button></>}
            <button type="button" className="h-8 rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)]" disabled={busy || !check.detail || check.result !== null} onClick={runCheck}>{o.runCheck}</button>
            {check.result === true && <button type="button" className="h-8 rounded-sm border-none bg-[var(--color-success)] px-4 text-[13px] text-white" disabled={busy} onClick={reserveFromCheck}>{o.continueReserve}</button>}
            <button type="button" className="h-8 rounded-sm border border-[var(--shell-input-border)] px-4 text-[13px]" onClick={() => setCheck(null)}>{t.pages.company.cancel}</button>
          </div>}>
          <div className="flex flex-col gap-4 p-4 text-[13px]">
            <div><div className="font-medium text-[var(--shell-heading)]">{check.row.orderNo}</div><div className="text-[var(--shell-crumb-text)]">{check.row.customer} · {check.row.product} · {check.row.address}</div></div>
            {check.detail && <>
              <div className="grid grid-cols-4 gap-2 text-center"><div>{o.totalPorts}<b className="block text-lg">{check.detail.total}</b></div><div>{o.idlePorts}<b className="block text-lg text-[var(--color-success)]">{check.detail.idle}</b></div><div>{o.reservedPorts}<b className="block text-lg">{check.detail.reserved}</b></div><div>{o.usedPorts}<b className="block text-lg">{check.detail.used}</b></div></div>
              <table className="w-full border-collapse"><thead><tr><th className="p-2 text-left">{o.device}</th><th className="p-2 text-left">{o.deviceStatus}</th><th className="p-2 text-right">{o.idlePorts}</th></tr></thead><tbody>{check.detail.devices.map((d) => <tr key={d.code}><td className="border-t p-2">{d.name || d.code}</td><td className="border-t p-2">{d.status}</td><td className="border-t p-2 text-right">{d.idle}/{d.total}</td></tr>)}</tbody></table>
              <div><div className="mb-1">{o.idlePortCodes}</div><div className="flex flex-wrap gap-1">{check.detail.idleCodes.map((code) => <span key={code} className="rounded bg-[var(--shell-menu-hover-bg)] px-2 py-1">{code}</span>)}</div></div>
            </>}
            {check.result === true && <div className="rounded border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] p-3 text-[var(--color-success)]">{check.message}</div>}
            {check.result === false && <div className="rounded border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] p-3 text-[var(--color-danger)]">{check.message}<div className="mt-1 text-[var(--shell-content-text)]">{o.failureHelp}</div></div>}
            {check.message && check.result === null && <div className="text-[var(--color-danger)]">{check.message}</div>}
          </div>
        </Drawer>
      )}
      {(track || trackError) && (
      <Drawer title={o.trackTitle} onClose={() => { setTrack(null); setTrackError('') }}
        footer={<button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setTrack(null)}>
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
                  <tr><td colSpan={5} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><EmptyState text={o.empty} /></td></tr>
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
