// 订单推进卡:所选客户订单(GET /orders?customerId=)逐单推进
// (核查→预占→收费→激活;取消需二次确认),并可发起新订单。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useConfirm } from '../../../components/ConfirmDialog'
import { StatusTag } from '../../../components/StatusTag'
import { DataTable, type ColumnDef } from '../../../components/business/data-table'
import { OrderCreateDrawer } from '../../boss/order/OrderCreateDrawer'
import type { OrderListRow } from '../../boss/types'
import { useT } from '../../../i18n'

type Action = 'check-resource' | 'reserve' | 'charge' | 'cancel' | 'activate'

export function OrderAdvanceCard({
  customerId, orders, onChanged,
}: { customerId: string; orders: OrderListRow[]; onChanged: () => void }) {
  const t = useT()
  const w = t.pages.onboardingPage
  const o = t.pages.orderPage
  const confirmDialog = useConfirm()
  const [createOpen, setCreateOpen] = useState(false)
  const [busyNo, setBusyNo] = useState('')
  const [msg, setMsg] = useState<{ no: string; ok: boolean; text: string } | null>(null)

  if (!customerId) return null

  const act = async (row: OrderListRow, action: Action) => {
    if (busyNo !== '') return
    if (action === 'cancel' && !(await confirmDialog(o.confirmCancel.replace('{no}', row.orderNo), { danger: true }))) return
    setBusyNo(row.orderNo); setMsg(null)
    try {
      const d = await apiFetch<{ available?: boolean; idlePorts?: string[]; stage?: number }>(`/orders/${encodeURIComponent(row.orderNo)}/${action}`, { method: 'POST' })
      if (action === 'check-resource') {
        const ok = d?.available === true
        setMsg({ no: row.orderNo, ok, text: ok ? o.checkPass.replace('{count}', String(d?.idlePorts?.length ?? 0)) : o.checkFail })
      } else {
        setMsg({ no: row.orderNo, ok: true, text: `${row.orderNo} → ${w.stageUnit.replace('{n}', String(d?.stage ?? row.stage))}` })
      }
      onChanged()
    } catch (e) {
      setMsg({ no: row.orderNo, ok: false, text: e instanceof Error ? e.message : o.actionFail })
    } finally { setBusyNo('') }
  }

  const nextAction = (r: OrderListRow): { key: Action | ''; label: string } => {
    if (r.status === 'PENDING' && r.stage === 1) return { key: 'check-resource', label: o.actCheck }
    if (r.status === 'PENDING' && r.stage === 2) return { key: 'reserve', label: o.actReserve }
    if (r.status === 'RESERVED' && r.stage === 3) return { key: 'charge', label: o.actCharge }
    if (r.status === 'INSTALLING' && r.stage === 9) return { key: 'activate', label: w.activate }
    return { key: '', label: r.status === 'INSTALLING' ? w.waitWorker : '—' }
  }

  const columns: ColumnDef[] = [
    { key: 'orderNo', label: w.orderColumns[0] },
    { key: 'product', label: w.orderColumns[1] },
    { key: 'stageLabel', label: w.orderColumns[2], render: (r) => `${r.stage}. ${r.stageLabel}` },
    { key: 'status', label: w.orderColumns[3], render: (r) => <StatusTag domain="order" value={String(r.status)} /> },
    {
      key: 'ops',
      label: w.orderColumns[4],
      render: (r) => {
        const row = r as unknown as OrderListRow
        const next = nextAction(row)
        const busy = busyNo === row.orderNo
        return (
          <span className="inline-flex items-center gap-3">
            {next.key !== '' && (
              <button type="button" disabled={busy} onClick={() => act(row, next.key as Exclude<Action, 'cancel'>)}>{busy ? t.common.loading : next.label}</button>
            )}
            {next.key === '' && <span className="text-[var(--shell-group-title)]">{next.label}</span>}
            {row.status !== 'DONE' && row.status !== 'CANCELLED' && (
              <>
                <span className="text-[var(--shell-side-border)]">|</span>
                <button type="button" disabled={busy} onClick={() => act(row, 'cancel')}>{o.actCancel}</button>
              </>
            )}
          </span>
        )
      },
    },
  ]

  return (
    <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <span className="text-[13px] font-medium text-[var(--shell-heading)]">4. {w.orderTitle}</span>
        <span className="spacer" />
        <button type="button" className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setCreateOpen(true)}>{w.orderBtn}</button>
      </div>
      <div className="px-4 pb-4">
        <DataTable columns={columns} rows={orders as unknown as Record<string, unknown>[]} emptyText={w.emptyOrders} />
      </div>
      {msg && (
        <div className={'mx-4 mb-3 rounded-sm border px-3 py-2 text-[13px] ' + (msg.ok
          ? 'border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] text-[var(--color-success)]'
          : 'border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] text-[var(--color-danger)]')}>
          {msg.no} · {msg.text}
        </div>
      )}
      {createOpen && (
        <OrderCreateDrawer open fixedCustomerId={customerId} onClose={() => setCreateOpen(false)} onCreated={onChanged} />
      )}
    </div>
  )
}
