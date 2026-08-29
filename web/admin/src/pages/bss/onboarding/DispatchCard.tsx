// 派单卡:所选客户订单的待派工单(GET /dispatch/pool 按 orderId 匹配)
// + WorkerPicker 检索指派(POST /dispatch/pool/:ticketNo/assign)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { WorkerPicker, type PickedWorker } from '../../boss/dispatch/WorkerPicker'
import { ErrorBanner } from '../../../components/business/page-head'
import type { DispatchTicketRow } from '../../boss/types'
import { useT } from '../../../i18n'

export function DispatchCard({
  orders, onChanged,
}: { orders: OrderListRowLite[]; onChanged: () => void }) {
  const t = useT()
  const w = t.pages.onboardingPage
  const [tickets, setTickets] = useState<DispatchTicketRow[]>([])
  const [picked, setPicked] = useState<PickedWorker | null>(null)
  const [busyNo, setBusyNo] = useState('')
  const [msg, setMsg] = useState<{ no: string; ok: boolean; text: string } | null>(null)
  const [error, setError] = useState('')

  const orderIds = new Set(orders.map((o) => o.id))
  const refresh = () => {
    apiFetch<{ items: DispatchTicketRow[] }>('/dispatch/pool')
      .then((d) => setTickets((d?.items ?? []).filter((tk) => orderIds.has(tk.orderId))))
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
  }
  useEffect(() => {
    if (!orders.length) { setTickets([]); return }
    refresh()
  }, [orders]) // eslint-disable-line react-hooks/exhaustive-deps

  const assign = async (tk: DispatchTicketRow) => {
    if (busyNo !== '' || !picked) return
    setBusyNo(tk.ticketNo); setMsg(null); setError('')
    try {
      await apiFetch(`/dispatch/pool/${encodeURIComponent(tk.ticketNo)}/assign`, {
        method: 'POST', body: { masterId: picked.id },
      })
      setMsg({ no: tk.ticketNo, ok: true, text: w.assignedTo.replace('{name}', picked.name) })
      setPicked(null)
      onChanged()
    } catch (e) {
      setMsg({ no: tk.ticketNo, ok: false, text: e instanceof Error ? e.message : w.actionFail })
    } finally { setBusyNo('') }
  }

  if (!orders.length) return null
  return (
    <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="text-[13px] font-medium text-[var(--shell-heading)]">5. {w.dispatchTitle}</span>
        <span className="spacer" />
      </div>
      {error && <ErrorBanner message={error} className="mx-4 mb-3" />}
      {msg && (
        <div className={'mx-4 mb-3 rounded-sm border px-3 py-2 text-[13px] ' + (msg.ok
          ? 'border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] text-[var(--color-success)]'
          : 'border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] text-[var(--color-danger)]')}>
          {msg.no} · {msg.text}
        </div>
      )}
      {tickets.length ? (
        <div className="px-4 pb-4">
          {tickets.map((tk) => (
            <div key={tk.ticketNo} className="mb-3 rounded-md border border-[var(--shell-side-border)] p-3">
              <div className="mb-2 flex flex-wrap items-center gap-2 text-[13px]">
                <span className="font-medium text-[var(--shell-heading)]">{tk.ticketNo}</span>
                <span className="text-[var(--shell-group-title)]">{tk.regionName || '—'} · {tk.status}</span>
              </div>
              <WorkerPicker selectedId={picked ? String(picked.id) : ''} onSelect={setPicked} />
              <div className="mt-2 flex justify-end">
                <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50"
                  disabled={busyNo === tk.ticketNo || !picked} onClick={() => assign(tk)}>{w.assign}</button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="px-4 pb-4 text-[13px] text-[var(--shell-group-title)]">{w.noTicket}</div>
      )}
    </div>
  )
}

interface OrderListRowLite {
  id: number
  status: string
}
