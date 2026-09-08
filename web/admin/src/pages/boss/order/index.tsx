// 订单管理页:契约 GET /orders(keyword/status 过滤);跟踪抽屉 GET /orders/:orderNo(order+timeline);
// 环节推进 POST /orders/:orderNo/{check-resource,reserve,charge,cancel}(order_workflow.go);
// 代客下单 POST /orders(线下受理场景)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useNavigate } from 'react-router-dom'
import { useQueryInt, useQueryState } from '../../../lib/useQueryState'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Dropdown } from '../../../components/Dropdown'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { LoadingState, TableStateRow } from '../../../components/business'
import { bizDateKey, fmtTime } from '../../../lib/format'
import { createdAtText, timelineFinishedText } from './timeCells'
import { pageSlice, type CheckDetail, type OrderListRow, type TimelineRow, type WorkerLocationRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { OrderCreateDrawer } from './OrderCreateDrawer'

const STATUSES = ['PENDING', 'RESERVED', 'INSTALLING', 'DONE', 'CANCELLED'] as const

/** 行内动作链接:busy 提交中禁用,视觉保持行内链接形态。 */
function RowAction({ label, disabled, onClick }: { label: string; disabled?: boolean; onClick: () => void }) {
  return (
    <button type="button" disabled={disabled} onClick={onClick}
      className="cursor-pointer border-none bg-none px-0 text-xs text-[var(--color-text-link)] hover:underline disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:no-underline">
      {label}
    </button>
  )
}

export default function OrderPage() {
  const t = useT()
  const nav = useNavigate()
  const confirmDialog = useConfirm()
  const o = t.pages.orderPage
  const [rows, setRows] = useState<OrderListRow[]>([])
  const [error, setError] = useState('')
  const [urlKeyword, setUrlKeyword] = useQueryState('kw', '')
  const [urlStatus, setUrlStatus] = useQueryState('status', '')
  const [urlCreated, setUrlCreated] = useQueryState('created', '')
  const [urlPage, setUrlPage] = useQueryInt('page', 1)
  const [urlPageSize, setUrlPageSize] = useQueryInt('size', 10)
  const [keyword, setKeyword] = useState(urlKeyword)
  const [status, setStatus] = useState(urlStatus)
  const created = urlCreated
  const [page, setPage] = useState(urlPage)
  const [pageSize, setPageSize] = useState(urlPageSize)
  const [busy, setBusy] = useState(false)
  const [track, setTrack] = useState<{ order: OrderListRow; timeline: TimelineRow[]; latestLocation: WorkerLocationRow | null } | null>(null)
  const [trackOpen, setTrackOpen] = useState(false)
  const [trackError, setTrackError] = useState('')
  const [check, setCheck] = useState<{ row: OrderListRow; detail: CheckDetail | null; result: boolean | null; message: string } | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  // override:状态下拉等触发时状态还没进闭包,显式带参避免旧值过滤(102 实测筛选不生效)。
  const load = (override?: { keyword?: string; status?: string }) => {
    const kw = override?.keyword ?? keyword
    const st = override?.status ?? status
    setError('')
    setBusy(true)
    apiFetch<{ items: OrderListRow[] }>('/orders', {
      query: { keyword: kw || undefined, status: st || undefined, created: created || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : o.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openTrack = (orderNo: string) => {
    setTrackOpen(true) // 先开抽屉再取数,加载态立即响应点击
    setTrack(null)
    setTrackError('')
    apiFetch<{ order: OrderListRow; timeline: TimelineRow[]; latestLocation: WorkerLocationRow | null }>(`/orders/${encodeURIComponent(orderNo)}`)
      .then((d) => setTrack({ order: d?.order ?? null as unknown as OrderListRow, timeline: d?.timeline ?? [], latestLocation: d?.latestLocation ?? null }))
      .catch((e) => setTrackError(e instanceof Error ? e.message : o.trackFail))
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
      .then(() => { setCheck(null); toast.success(o.toastReserveOk.replace('{no}', check.row.orderNo)); load() })
      .catch((e) => setCheck((x) => x ? { ...x, message: e instanceof Error ? e.message : o.actionFail } : x))
      .finally(() => setBusy(false))
  }

  const advance = async (row: OrderListRow, action: 'reserve' | 'charge' | 'cancel') => {
    if (busy) return
    if (action === 'cancel' && !(await confirmDialog(o.confirmCancel.replace('{no}', row.orderNo), { danger: true }))) return
    setBusy(true)
    setError('')
    const okMsg = (action === 'reserve' ? o.toastReserveOk : action === 'charge' ? o.toastChargeOk : o.toastCancelOk).replace('{no}', row.orderNo)
    apiFetch(`/orders/${encodeURIComponent(row.orderNo)}/${action}`, { method: 'POST' })
      .then(() => { toast.success(okMsg); load() })
      .catch((e) => {
        const msg = e instanceof Error ? e.message : o.actionFail
        setError(msg)
        toast.error(msg) // 行内动作失败即时反馈;横幅留页面级可复制原因
      })
      .finally(() => setBusy(false))
  }

  const statusOptions = [{ value: '', label: o.allStatus }, ...STATUSES.map((value, i) => ({ value, label: o.statusOptions[i] }))]
  const resetPage = () => { setPage(1); setUrlPage(1) }

  const updateKeyword = (value: string) => {
    setKeyword(value)
    setUrlKeyword(value)
    resetPage()
  }

  const updateStatus = (value: string) => {
    setStatus(value)
    setUrlStatus(value)
    resetPage()
    load({ status: value }) // 筛选即查,免去再点一次刷新
  }

  const toggleToday = () => {
    setUrlCreated(created ? '' : 'today')
    resetPage()
  }

  const updatePageSize = (value: number) => {
    setPageSize(value)
    setUrlPageSize(value)
    resetPage()
  }

  const todayRows = created === 'today'
    ? rows.filter((row) => bizDateKey(row.createdAt) === bizDateKey(new Date()))
    : rows
  const slice = pageSlice(todayRows, page, pageSize)

  return (
    <div>
      <PageHead title={o.title} desc={o.desc} />
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <Input className="w-60" placeholder={o.searchPlaceholder} value={keyword}
            onChange={(e) => updateKeyword(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') load() }} />
          <Dropdown value={status} options={statusOptions} onChange={updateStatus} ariaLabel={o.allStatus} />
          <ToolbarButton primary={created === 'today'} onClick={toggleToday}>{o.actToday}</ToolbarButton>
          <span className="spacer" />
          <ToolbarButton primary onClick={() => setCreateOpen(true)}>{o.createBtn}</ToolbarButton>
          <ToolbarButton disabled={busy} onClick={() => load()}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="px-4 pb-4">
          <Table>
            <TableHeader>
              <TableRow>{o.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.orderNo}>
                  <TableCell>{r.orderNo}</TableCell>
                  <TableCell>{r.customer || '—'}</TableCell>
                  <TableCell>{r.product || '—'}</TableCell>
                  <TableCell>{r.address || '—'}</TableCell>
                  <TableCell>{r.stage}. {r.stageLabel}</TableCell>
                  <TableCell><StatusTag domain="order" value={r.status} /></TableCell>
                  <TableCell>{createdAtText(r.createdAt)}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-3">
                      {r.stage === 1 && r.status === 'PENDING' && <RowAction label={o.actCheck} disabled={busy} onClick={() => openCheck(r)} />}
                      {r.stage === 2 && r.status === 'PENDING' && <RowAction label={o.actReserve} disabled={busy} onClick={() => openCheck(r)} />}
                      {r.stage === 3 && r.status === 'RESERVED' && <RowAction label={o.actCharge} disabled={busy} onClick={() => advance(r, 'charge')} />}
                      {r.status !== 'DONE' && r.status !== 'CANCELLED' && <RowAction label={o.actCancel} disabled={busy} onClick={() => advance(r, 'cancel')} />}
                      <RowAction label={o.track} onClick={() => openTrack(r.orderNo)} />
                    </span>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={8} loading={busy} text={o.empty} />}
            </TableBody>
          </Table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={todayRows.length} page={page} pageSize={pageSize}
            onPage={(value) => { setPage(value); setUrlPage(value) }}
            onSize={updatePageSize} {...pagerTexts(o)} />
        </div>
      </Card>
      {check && (
        <Drawer title={o.checkTitle} onClose={() => setCheck(null)}
          footer={<div className="flex gap-2">
            {check.result === false && <><ToolbarButton onClick={() => nav('/oss/expand')}>{o.expand}</ToolbarButton><ToolbarButton onClick={() => nav('/oss/transfer')}>{o.transfer}</ToolbarButton></>}
            <ToolbarButton primary disabled={busy || !check.detail || check.result !== null} onClick={runCheck}>{o.runCheck}</ToolbarButton>
            {check.result === true && <ToolbarButton disabled={busy} onClick={reserveFromCheck}>{o.continueReserve}</ToolbarButton>}
            <ToolbarButton onClick={() => setCheck(null)}>{t.pages.company.cancel}</ToolbarButton>
          </div>}>
          <div className="flex flex-col gap-4 p-4 text-[13px]">
            <div><div className="font-medium text-[var(--shell-heading)]">{check.row.orderNo}</div><div className="text-[var(--shell-crumb-text)]">{check.row.customer} · {check.row.product} · {check.row.address}</div></div>
            {check.detail && <>
              <div className="grid grid-cols-4 gap-2 text-center"><div>{o.totalPorts}<b className="block text-lg">{check.detail.total}</b></div><div>{o.idlePorts}<b className="block text-lg text-[var(--color-success)]">{check.detail.idle}</b></div><div>{o.reservedPorts}<b className="block text-lg">{check.detail.reserved}</b></div><div>{o.usedPorts}<b className="block text-lg">{check.detail.used}</b></div></div>
              <Table>
                <TableHeader><TableRow><TableHead>{o.device}</TableHead><TableHead>{o.deviceStatus}</TableHead><TableHead className="text-right">{o.idlePorts}</TableHead></TableRow></TableHeader>
                <TableBody>{check.detail.devices.map((d) => <TableRow key={d.code}><TableCell>{d.name || d.code}</TableCell><TableCell>{d.status}</TableCell><TableCell className="text-right">{d.idle}/{d.total}</TableCell></TableRow>)}</TableBody>
              </Table>
              <div><div className="mb-1">{o.idlePortCodes}</div><div className="flex flex-wrap gap-1">{check.detail.idleCodes.map((code) => <span key={code} className="rounded bg-[var(--shell-menu-hover-bg)] px-2 py-1">{code}</span>)}</div></div>
            </>}
            {check.result === true && <div className="rounded border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] p-3 text-[var(--color-success)]">{check.message}</div>}
            {check.result === false && <div className="rounded border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] p-3 text-[var(--color-danger)]">{check.message}<div className="mt-1 text-[var(--shell-content-text)]">{o.failureHelp}</div></div>}
            {check.message && check.result === null && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] break-all text-[var(--color-danger)]">{check.message}</div>}
          </div>
        </Drawer>
      )}
      {trackOpen && (
        <Drawer title={o.trackTitle} onClose={() => setTrackOpen(false)}
          footer={<ToolbarButton onClick={() => setTrackOpen(false)}>{t.pages.company.cancel}</ToolbarButton>}>
          {trackError ? <ErrorBanner message={trackError} /> : !track ? <LoadingState /> : (
            <>
              <div className="mx-4 mb-4 rounded-sm border border-[var(--shell-card-border)] p-3 text-[13px]">
                <div className="mb-2 font-medium">{o.locationTitle}</div>
                {track.latestLocation
                  ? <div>{o.locLat} {track.latestLocation.lat.toFixed(6)} · {o.locLng} {track.latestLocation.lng.toFixed(6)} · {o.locationUpdated} {fmtTime(track.latestLocation.reportedAt)}</div>
                  : <div>{o.locationEmpty}</div>}
              </div>
              <div className="px-4 pb-4">
                <Table>
                  <TableHeader>
                    <TableRow>{o.timelineColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
                  </TableHeader>
                  <TableBody>
                    {track.timeline.map((x) => (
                      <TableRow key={x.stage}>
                        <TableCell>{x.stage}. {x.name}</TableCell>
                        <TableCell>{timelineFinishedText(x.finishedAt, o.timelineUnfinished)}</TableCell>
                        <TableCell>{x.duration || '—'}</TableCell>
                        <TableCell>{x.retries}</TableCell>
                        <TableCell><StatusTag domain="ticket" value={x.result} /></TableCell>
                      </TableRow>
                    ))}
                    {!track.timeline.length && <TableStateRow colSpan={5} text={o.empty} />}
                  </TableBody>
                </Table>
              </div>
            </>
          )}
        </Drawer>
      )}
      {createOpen && (
        <OrderCreateDrawer open onClose={() => setCreateOpen(false)} onCreated={(no) => { if (no) toast.success(o.toastCreateOk.replace('{no}', no)); load() }} />
      )}
    </div>
  )
}
