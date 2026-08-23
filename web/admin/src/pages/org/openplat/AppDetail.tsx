// 应用详情:Webhook 订阅 CRUD + 测试事件 + 投递 outbox(死信 requeue)。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { formatTime } from '../../base/audit/logic'
import { EmptyState } from '../../../components/business'
import type { OpenAppRow } from './index'

interface SubRow {
  id: number
  eventType: string
  endpointUrl: string
  status: number
  createdAt: string
}

interface DeliveryRow {
  id: number
  subscriptionId: number
  eventId: string
  eventType: string
  status: number
  attempts: number
  httpStatus: number
  lastError: string
  createdAt: string
}

// filterAppDeliveries 只保留属于给定订阅集的投递(应用详情视图)。
export function filterAppDeliveries(deliveries: DeliveryRow[], subs: SubRow[]): DeliveryRow[] {
  const subIds = new Set(subs.map((s) => s.id))
  return deliveries.filter((d) => subIds.has(d.subscriptionId))
}

export function AppDetail({ app, onRefreshApps }: { app: OpenAppRow; onRefreshApps: () => void }) {
  const t = useT()
  const [subs, setSubs] = useState<SubRow[]>([])
  const [deliveries, setDeliveries] = useState<DeliveryRow[]>([])
  const [error, setError] = useState('')
  const [evType, setEvType] = useState('order.stage.done')
  const [endpoint, setEndpoint] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: SubRow[] }>(`/openplat/apps/${app.id}/subscriptions`)
      .then((d) => setSubs(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : t.pages.openplat.loadFail))
    apiFetch<{ items: DeliveryRow[] }>('/openplat/deliveries')
      .then((d) => setDeliveries(d?.items ?? []))
      .catch(() => setDeliveries([]))
  }, [app.id, t])
  useEffect(load, [load]) // eslint-disable-line react-hooks/exhaustive-deps

  const addSub = async () => {
    if (busy || !evType.trim() || !endpoint.trim()) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/apps/${app.id}/subscriptions`, {
        method: 'POST',
        body: JSON.stringify({ eventType: evType.trim(), endpointUrl: endpoint.trim() }),
      })
      setEndpoint('')
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.openplat.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const delSub = async (id: number) => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/subscriptions/${id}`, { method: 'DELETE' })
      load()
    } finally {
      setBusy(false)
    }
  }

  const testEvent = async () => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/apps/${app.id}/test-event`, { method: 'POST' })
      load()
      onRefreshApps()
    } catch (e) {
      setError(e instanceof Error ? e.message : t.pages.openplat.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const requeue = async (id: number) => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/deliveries/${id}/requeue`, { method: 'POST' })
      load()
    } finally {
      setBusy(false)
    }
  }

  const inputCls = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
  const appDeliveries = filterAppDeliveries(deliveries, subs)
  const statusLabel = (s: number) => s === 1 ? t.pages.openplat.dlvDone : s === 2 ? t.pages.openplat.dlvDead : t.pages.openplat.dlvPending

  return (
    <div className="mx-4 mb-6 rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-4">
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span className="text-sm font-medium text-[var(--shell-heading)]">{app.name} · {app.appId}</span>
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] hover:border-[var(--color-border-hover)]" disabled={busy} onClick={testEvent}>{t.pages.openplat.testEvent}</button>
      </div>

      <div className="mb-2 text-xs font-medium text-[var(--shell-group-title)]">{t.pages.openplat.subsTitle}</div>
      <div className="mb-2 flex flex-wrap gap-2">
        <input className={inputCls + ' w-44'} value={evType} placeholder="order.stage.done" onChange={(e) => setEvType(e.target.value)} />
        <input className={inputCls + ' flex-1 min-w-[240px]'} value={endpoint} placeholder={t.pages.openplat.pEndpoint} onChange={(e) => setEndpoint(e.target.value)} />
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={addSub}>{t.pages.openplat.addSub}</button>
      </div>
      {subs.length === 0 ? <EmptyState text={t.pages.openplat.noSubs} /> : (
        <table className="mb-4 w-full border-collapse text-[13px]">
          <tbody>
            {subs.map((s) => (
              <tr key={s.id}>
                <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono text-xs">{s.eventType}</td>
                <td className="h-9 px-2 break-all border-b border-[var(--shell-side-border)]">{s.endpointUrl}</td>
                <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{formatTime(s.createdAt)}</td>
                <td className="h-9 px-2 border-b border-[var(--shell-side-border)]"><button disabled={busy} onClick={() => delSub(s.id)}>{t.pages.openplat.delSub}</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className="mb-2 text-xs font-medium text-[var(--shell-group-title)]">{t.pages.openplat.dlvTitle}</div>
      {appDeliveries.length === 0 ? <EmptyState text={t.pages.openplat.noDeliveries} /> : (
        <table className="w-full border-collapse text-[13px]">
          <tbody>
            {appDeliveries.slice(0, 20).map((d) => (
              <tr key={d.id}>
                <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono text-xs">{d.eventType}</td>
                <td className="h-9 px-2 break-all border-b border-[var(--shell-side-border)] font-mono text-xs">{d.eventId}</td>
                <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{statusLabel(d.status)} · {d.attempts}</td>
                <td className="h-9 px-2 border-b border-[var(--shell-side-border)]">{d.lastError || (d.httpStatus ? `HTTP ${d.httpStatus}` : '')}</td>
                <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{d.status !== 1 && <button disabled={busy} onClick={() => requeue(d.id)}>{t.pages.openplat.requeue}</button>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {error && <div className="mt-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
    </div>
  )
}
