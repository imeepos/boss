// 派单管理页:工单池指派 / 我的工单 / 改派台账(order.yaml /dispatch 段)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type DispatchTicketRow, type DispatchTransferRow } from '../types'

export default function DispatchPage() {
  const t = useT()
  const d = t.pages.dispatchPage
  const [tab, setTab] = useState<'pool' | 'mine' | 'transfers'>('pool')
  const [pool, setPool] = useState<DispatchTicketRow[]>([])
  const [mine, setMine] = useState<DispatchTicketRow[]>([])
  const [transfers, setTransfers] = useState<DispatchTransferRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [workerFilter, setWorkerFilter] = useState('')
  const [act, setAct] = useState<{ mode: 'assign' | 'transfer'; ticket: DispatchTicketRow } | null>(null)
  const [masterId, setMasterId] = useState('')
  const [reason, setReason] = useState('')
  const [formError, setFormError] = useState('')

  const load = (key: string) => {
    setError('')
    setBusy(true)
    const req = key === 'pool' ? apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/pool')
      : key === 'mine'
        ? apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/my-tickets', { query: { workerId: workerFilter || undefined } })
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

  const slice = pageSlice<DispatchTicketRow | DispatchTransferRow>(
    tab === 'pool' ? pool : tab === 'mine' ? mine : transfers, page, pageSize,
  )
  const count = tab === 'pool' ? pool.length : tab === 'mine' ? mine.length : transfers.length

  return (
    <div>
      <PageHead title={d.title} desc={d.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0', alignItems: 'center' }}>
          {(['pool', 'mine', 'transfers'] as const).map((key) => (
            <button key={key} onClick={() => { setTab(key); setPage(1) }}
              style={{
                padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none', border: 'none',
                borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
                color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
              }}>
              {key === 'pool' ? d.tabPool : key === 'mine' ? d.tabMine : d.tabTransfers}
            </button>
          ))}
          {tab === 'mine' && (
            <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" style={{ width: 180 }} placeholder={d.filterWorker}
              value={workerFilter} onChange={(e) => { setWorkerFilter(e.target.value); setPage(1) }} />
          )}
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => load(tab)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : tab !== 'transfers' ? (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{d.ticketColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(slice as DispatchTicketRow[]).map((x) => (
                  <tr key={x.ticketId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.ticketNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.orderId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.workerName || (x.workerId ? `#${x.workerId}` : '—')}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.groupName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.regionName || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="ticket" value={x.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        {x.workerId === 0 ? (
                          <button disabled={busy} onClick={() => { setAct({ mode: 'assign', ticket: x }); setMasterId(''); setFormError('') }}>
                            {d.assign}
                          </button>
                        ) : (
                          <button disabled={busy} onClick={() => { setAct({ mode: 'transfer', ticket: x }); setMasterId(''); setReason(''); setFormError('') }}>
                            {d.transfer}
                          </button>
                        )}
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{d.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{d.transferColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {(slice as DispatchTransferRow[]).map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.id}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.ticketId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.fromWorkerName || (x.fromWorkerId ? `#${x.fromWorkerId}` : '—')}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.toWorkerName || (x.toWorkerId ? `#${x.toWorkerId}` : '—')}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{x.reason || '—'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.transferredAt)}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{d.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={count} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(d)} />
        </div>
      </div>
      {act && (
        <Drawer title={act.mode === 'assign' ? d.assign : d.transfer} onClose={() => setAct(null)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setAct(null)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <div className="flex flex-col gap-1.5">
              <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.fMaster}({act.ticket.ticketNo})</label>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" value={masterId} placeholder={d.pMaster}
                onChange={(e) => setMasterId(e.target.value)} />
            </div>
            {act.mode === 'transfer' && (
              <div className="flex flex-col gap-1.5">
                <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{d.fReason}</label>
                <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={reason} placeholder={d.pReason}
                  onChange={(e) => setReason(e.target.value)} />
              </div>
            )}
            {formError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{formError}</div>}
          </div>
        </Drawer>
      )}
    </div>
  )
}
