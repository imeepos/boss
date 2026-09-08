// 派单卡:所选客户在途订单的工单状态驱动(GET /orders/:no 聚合 ticket)——
// 未指派→WorkerPicker 检索指派;已指派且环节9(扫码后)→激活(POST /tickets/:no/activate,
// 自动推进 11-12 → DONE);环节8 未扫码提示等待装维。扫码绑定本身归装维现场作业。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Card } from '../../../components/ui/card'
import { WorkerPicker, type PickedWorker } from '../../boss/dispatch/WorkerPicker'
import { ErrorBanner } from '../../../components/business/page-head'
import type { OrderListRow } from '../../boss/types'
import { useT } from '../../../i18n'

interface TicketInfo {
  ticketNo: string
  workerId: number
  workerName: string
  status: string
}
interface Row {
  order: OrderListRow
  ticket: TicketInfo | null
}

export function DispatchCard({
  orders, onChanged,
}: { orders: OrderListRow[]; onChanged: () => void }) {
  const t = useT()
  const w = t.pages.onboardingPage
  const confirmDialog = useConfirm()
  const [rows, setRows] = useState<Row[]>([])
  const [picked, setPicked] = useState<Record<string, PickedWorker>>({})
  const [busyNo, setBusyNo] = useState('')
  const [msg, setMsg] = useState<{ no: string; ok: boolean; text: string } | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    const active = orders.filter((o) => o.status === 'INSTALLING')
    if (!active.length) { setRows([]); return }
    let alive = true
    Promise.all(active.map(async (o): Promise<Row> => {
      const d = await apiFetch<{ ticket: TicketInfo | null }>(`/orders/${encodeURIComponent(o.orderNo)}`)
      return { order: o, ticket: d?.ticket ?? null }
    }))
      .then((rs) => { if (alive) setRows(rs) })
      .catch((e) => { if (alive) setError(e instanceof Error ? e.message : w.loadFail) })
    return () => { alive = false }
  }, [orders]) // eslint-disable-line react-hooks/exhaustive-deps

  const run = async (row: Row, action: 'assign' | 'activate', worker?: PickedWorker) => {
    const tk = row.ticket
    if (!tk || busyNo !== '') return
    if (action === 'assign' && !worker) { setError(w.pickWorkerFirst); return }
    if (action === 'activate' && !(await confirmDialog(w.activateConfirm.replace('{no}', row.order.orderNo)))) return
    setBusyNo(tk.ticketNo); setMsg(null); setError('')
    try {
      if (action === 'assign') {
        await apiFetch(`/dispatch/pool/${encodeURIComponent(tk.ticketNo)}/assign`, {
          method: 'POST', body: { masterId: worker!.id },
        })
        const text = w.assignedTo.replace('{name}', worker!.name)
        setMsg({ no: tk.ticketNo, ok: true, text })
        toast.success(text)
      } else {
        const d = await apiFetch<{ stage?: number }>(`/tickets/${encodeURIComponent(tk.ticketNo)}/activate`, { method: 'POST' })
        const text = `${row.order.orderNo} → ${w.stageUnit.replace('{n}', String(d?.stage ?? 12))}`
        setMsg({ no: tk.ticketNo, ok: true, text })
        toast.success(text)
      }
      setPicked((p) => { const n = { ...p }; delete n[tk.ticketNo]; return n })
      onChanged()
    } catch (e) {
      setMsg({ no: tk.ticketNo, ok: false, text: e instanceof Error ? e.message : w.actionFail })
    } finally { setBusyNo('') }
  }

  if (!orders.length) return null
  return (
    <Card>
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
      {rows.length ? (
        <div className="px-4 pb-4">
          {rows.map(({ order, ticket }) => (
            <div key={order.orderNo} className="mb-3 rounded-md border border-[var(--shell-side-border)] p-3">
              <div className="mb-2 flex flex-wrap items-center gap-2 text-[13px]">
                <span className="font-medium text-[var(--shell-heading)]">{ticket?.ticketNo || order.orderNo}</span>
                <span className="text-[var(--shell-group-title)]">{order.stage}. {order.stageLabel}</span>
                {ticket && ticket.workerId > 0 && <span className="text-[var(--color-success)]">{w.assignedTo.replace('{name}', ticket.workerName)}</span>}
              </div>
              {!ticket && <div className="text-[13px] text-[var(--shell-group-title)]">{w.notDispatched}</div>}
              {ticket && ticket.workerId === 0 && (
                <>
                  <WorkerPicker
                    selectedId={picked[ticket.ticketNo] ? String(picked[ticket.ticketNo].id) : ''}
                    onSelect={(wk) => setPicked((p) => ({ ...p, [ticket.ticketNo]: wk }))}
                  />
                  <div className="mt-2 flex justify-end">
                    <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50"
                      disabled={busyNo === ticket.ticketNo || !picked[ticket.ticketNo]} onClick={() => run({ order, ticket }, 'assign', picked[ticket.ticketNo])}>{w.assign}</button>
                  </div>
                </>
              )}
              {ticket && ticket.workerId > 0 && order.stage === 8 && <div className="text-[13px] text-[var(--shell-group-title)]">{w.waitScan}</div>}
              {ticket && ticket.workerId > 0 && order.stage === 9 && (
                <div className="flex justify-end">
                  <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50"
                    disabled={busyNo === ticket.ticketNo} onClick={() => run({ order, ticket }, 'activate')}>{w.activate}</button>
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="px-4 pb-4 text-[13px] text-[var(--shell-group-title)]">{w.noActiveTicket}</div>
      )}
    </Card>
  )
}
