// 派单管理页:工单池指派 / 我的工单 / 改派台账(order.yaml /dispatch 段)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useQueryState } from '../../../lib/useQueryState'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner, ToolbarButton, IdRef } from '../../../components/business'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type DispatchTicketRow, type DispatchTransferRow } from '../types'
import { WorkerPicker, type PickedWorker } from './WorkerPicker'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { searchWorkers } from '../../../api/pickers'
import { TableStateRow } from '../../../components/business'
import { TabBar } from '../../../components/business/tab-bar'
import { Card } from '../../../components/ui/card'
import { Input } from '../../../components/ui/input'
import { Button } from '../../../components/ui/button'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'

/** 行内动作链接:busy 提交中禁用。 */
function RowAction({ label, disabled, onClick }: { label: string; disabled?: boolean; onClick: () => void }) {
  return (
    <button type="button" disabled={disabled} onClick={onClick}
      className="cursor-pointer border-none bg-none px-0 text-xs text-[var(--color-text-link)] hover:underline disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:no-underline">
      {label}
    </button>
  )
}

export default function DispatchPage() {
  const t = useT()
  const d = t.pages.dispatchPage
  const [urlStatus] = useQueryState('status', '')
  const [tab, setTab] = useState<'pool' | 'mine' | 'transfers'>('pool')
  const [pool, setPool] = useState<DispatchTicketRow[]>([])
  const [mine, setMine] = useState<DispatchTicketRow[]>([])
  const [transfers, setTransfers] = useState<DispatchTransferRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [workerFilter, setWorkerFilter] = useState('')
  /** 已选师傅人读名缓存:页签切回/检索失败时钉选回显,避免触发器跌回裸编号(W0 契约 6)。 */
  const [workerNames, setWorkerNames] = useState<Map<number, string>>(new Map())
  const [act, setAct] = useState<{ mode: 'assign' | 'transfer'; ticket: DispatchTicketRow } | null>(null)
  const [masterId, setMasterId] = useState('')
  const [picked, setPicked] = useState<PickedWorker | null>(null)
  const [reason, setReason] = useState('')
  const [formError, setFormError] = useState('')

  // wf:师傅筛选触发时状态未进闭包,显式带参(与订单页同款 stale-closure 防线)。
  const load = (key: string, wf?: string) => {
    setError('')
    setBusy(true)
    const req = key === 'pool' ? apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/pool')
      : key === 'mine'
        ? apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/my-tickets', { query: { workerId: (wf ?? workerFilter) || undefined } })
        : apiFetch<{ items: DispatchTransferRow[] }>('/dispatch/transfers')
    req.then((x) => {
      const items = (x as { items?: DispatchTicketRow[] & DispatchTransferRow[] } | null)?.items ?? []
      if (key === 'pool') setPool(items as DispatchTicketRow[])
      else if (key === 'mine') setMine(items as DispatchTicketRow[])
      else setTransfers(items as DispatchTransferRow[])
    })
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load(tab) }, [tab]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    searchWorkers('')
      .then((list) => setWorkerNames(new Map((list ?? []).map((w) => [w.id, `${w.name} · ${w.staffNo}`]))))
      .catch(() => setWorkerNames(new Map()))
  }, [])

  const submit = async () => {
    if (!act || busy) return
    if (!/^\d+$/.test(masterId) || Number(masterId) <= 0) { setFormError(d.eInput); return }
    if (act.mode === 'transfer' && !reason.trim()) { setFormError(d.eInput); return }
    setBusy(true)
    setFormError('')
    try {
      if (act.mode === 'assign') {
        await apiFetch(`/dispatch/pool/${encodeURIComponent(act.ticket.ticketNo)}/assign`, {
          method: 'POST', body: { masterId: Number(masterId) },
        })
      } else {
        await apiFetch(`/dispatch/tickets/${encodeURIComponent(act.ticket.ticketNo)}/transfer`, {
          method: 'POST', body: { toMasterId: Number(masterId), reason: reason.trim() },
        })
      }
      toast.success((act.mode === 'assign' ? d.toastAssignOk : d.toastTransferOk).replace('{no}', act.ticket.ticketNo))
      setAct(null)
      setMasterId('')
      setReason('')
      load('pool')
      load('mine')
      load('transfers')
    } catch (e) {
      setFormError(e instanceof Error ? e.message : d.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const source = tab === 'pool' ? pool : tab === 'mine' ? mine : transfers
  const filtered = urlStatus
    ? source.filter((row) => 'status' in row && row.status === urlStatus)
    : source
  const slice = pageSlice<DispatchTicketRow | DispatchTransferRow>(filtered, page, pageSize)
  const count = filtered.length

  const updateWorkerFilter = (v: string) => {
    setWorkerFilter(v)
    setPage(1)
    load('mine', v) // 筛选即查
  }

  const ticketHead = (
    <TableHeader>
      <TableRow>{d.ticketColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
    </TableHeader>
  )
  const transferHead = (
    <TableHeader>
      <TableRow>{d.transferColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
    </TableHeader>
  )

  return (
    <div>
      <PageHead title={d.title} desc={d.desc} />
      <Card>
        <TabBar
          tabs={[{ key: 'pool', label: d.tabPool }, { key: 'mine', label: d.tabMine }, { key: 'transfers', label: d.tabTransfers }]}
          value={tab}
          onChange={(k) => { setTab(k); setPage(1) }}
          extra={
            <div className="flex items-center gap-2 px-2">
          {tab === 'mine' && (
            <ResourcePicker
              value={workerFilter}
              onChange={updateWorkerFilter}
              search={searchWorkers}
              toOption={(w) => ({ value: String(w.id), label: `${w.name} · ${w.staffNo}` })}
              ariaLabel={d.filterWorker}
              emptyLabel={t.pages.pickers.common.all}
              searchPlaceholder={t.pages.pickers.common.placeholder}
              errorText={d.loadFail}
              pinnedOptions={
                workerFilter && workerNames.get(Number(workerFilter))
                  ? [{ value: workerFilter, label: workerNames.get(Number(workerFilter))! }]
                  : undefined
              }
            />
              )}
              <ToolbarButton disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</ToolbarButton>
            </div>
          }
        />
        {error && <ErrorBanner message={error} />}
        {tab !== 'transfers' ? (
          <div className="px-4 pb-4">
            <Table>
              {ticketHead}
              <TableBody>
                {(slice as DispatchTicketRow[]).map((x) => (
                  <TableRow key={x.ticketId}>
                    <TableCell>{x.ticketNo}</TableCell>
                    <TableCell><IdRef value={x.orderId} /></TableCell>
                    <TableCell>{x.workerName || (x.workerId ? `#${x.workerId}` : '—')}</TableCell>
                    <TableCell>{x.groupName || '—'}</TableCell>
                    <TableCell>{x.regionName || '—'}</TableCell>
                    <TableCell><StatusTag domain="ticket" value={x.status} /></TableCell>
                    <TableCell>
                      {x.workerId === 0 ? (
                        <RowAction label={d.assign} disabled={busy} onClick={() => { setAct({ mode: 'assign', ticket: x }); setMasterId(''); setPicked(null); setFormError('') }} />
                      ) : (
                        <RowAction label={d.transfer} disabled={busy} onClick={() => { setAct({ mode: 'transfer', ticket: x }); setMasterId(''); setPicked(null); setReason(''); setFormError('') }} />
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={d.empty} />}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className="px-4 pb-4">
            <Table>
              {transferHead}
              <TableBody>
                {(slice as DispatchTransferRow[]).map((x) => (
                  <TableRow key={x.id}>
                    <TableCell><IdRef value={x.id} /></TableCell>
                    <TableCell><IdRef value={x.ticketId} /></TableCell>
                    <TableCell>{x.fromWorkerName || (x.fromWorkerId ? `#${x.fromWorkerId}` : '—')}</TableCell>
                    <TableCell>{x.toWorkerName || (x.toWorkerId ? `#${x.toWorkerId}` : '—')}</TableCell>
                    <TableCell>{x.reason || '—'}</TableCell>
                    <TableCell>{fmtTime(x.transferredAt)}</TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={d.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={count} page={page} pageSize={pageSize}
            onPage={setPage} onSize={(s) => { setPageSize(s); setPage(1) }} {...pagerTexts(d)} />
        </div>
      </Card>
      {act && (
        <Drawer title={act.mode === 'assign' ? d.assign : d.transfer} onClose={() => setAct(null)}
          footer={
            <>
              <Button variant="outline" size="sm" onClick={() => setAct(null)}>{t.pages.company.cancel}</Button>
              <Button size="sm" disabled={busy} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </Button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.fMaster}({act.ticket.ticketNo})</label>
              <div><WorkerPicker selectedId={masterId} onSelect={(w) => { setMasterId(String(w.id)); setPicked(w); setFormError('') }} /></div>
              {picked && (
                <div className="text-xs text-[var(--shell-crumb-text)]">
                  {picked.name} · {picked.staffNo} · {picked.phone || '—'} · {picked.groupName} · {picked.regionName}
                </div>
              )}
            </div>
            {act.mode === 'transfer' && (
              <div className="flex flex-col gap-1.5">
                <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.fReason}</label>
                <Input value={reason} placeholder={d.pReason} onChange={(e) => setReason(e.target.value)} />
              </div>
            )}
            {formError && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] break-all text-[var(--color-danger)]">{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
