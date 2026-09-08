// 应用详情:Webhook 订阅 CRUD(抽屉式新增) + 测试事件 + 投递 outbox(死信 requeue)。
// 渲染为右侧 Drawer + 顶栏 TabBar(订阅/投递),订阅默认在前,投递独立 tab,避免被卡片底部挤出视野。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useConfirm } from '../../../components/ConfirmDialog'
import { formatTime } from '../../base/audit/logic'
import { EmptyState, TabBar } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Drawer } from '../../../components/Drawer'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { SubForm } from './SubForm'
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

// 详情抽屉内 tab key(稳定字符串,避免与业务 id 混淆)。
type DetailTab = 'subs' | 'deliveries'

export function AppDetail({ app, onRefreshApps, onClose }: {
  app: OpenAppRow
  onRefreshApps: () => void
  onClose: () => void
}) {
  const t = useT()
  const confirmDialog = useConfirm()
  const [tab, setTab] = useState<DetailTab>('subs')
  const [subs, setSubs] = useState<SubRow[]>([])
  const [deliveries, setDeliveries] = useState<DeliveryRow[]>([])
  const [error, setError] = useState('')
  const [subOpen, setSubOpen] = useState(false)
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

  const closeSubForm = () => setSubOpen(false)

  const delSub = async (id: number) => {
    if (busy) return
    if (!(await confirmDialog(t.pages.openplat.delSubConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/subscriptions/${id}`, { method: 'DELETE' })
      toast.success(t.pages.openplat.delSubOk)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t.pages.openplat.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const testEvent = async () => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/apps/${app.id}/test-event`, { method: 'POST' })
      toast.success(t.pages.openplat.testOk)
      load()
      onRefreshApps()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t.pages.openplat.saveFail)
    } finally {
      setBusy(false)
    }
  }

  const requeue = async (id: number) => {
    if (busy) return
    setBusy(true)
    try {
      await apiFetch(`/openplat/deliveries/${id}/requeue`, { method: 'POST' })
      toast.success(t.pages.openplat.requeueOk)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t.pages.openplat.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const appDeliveries = filterAppDeliveries(deliveries, subs)
  const statusLabel = (s: number) => s === 1 ? t.pages.openplat.dlvDone : s === 2 ? t.pages.openplat.dlvDead : t.pages.openplat.dlvPending
  const appStatusLabel = app.status === 1 ? t.pages.openplat.active : t.pages.openplat.disabled
  const appStatusClass = app.status === 1
    ? 'bg-[color-mix(in_srgb,var(--color-success)_15%,transparent)] text-[var(--color-success)]'
    : 'bg-[color-mix(in_srgb,var(--color-warning)_15%,transparent)] text-[var(--color-warning)]'

  return (
    <Drawer
      title={app.name}
      onClose={onClose}
      width={760}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={testEvent}>{t.pages.openplat.testEvent}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={() => setSubOpen(true)}>+ {t.pages.openplat.addSub}</button>
        </>
      }
    >
      <div className="mb-4 flex flex-wrap items-center gap-3 rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] px-4 py-3 text-[13px]">
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--shell-group-title)]">{t.pages.openplat.appIdLabel}</span>
          <code className="font-mono text-[12px] text-[var(--shell-content-text)]">{app.appId}</code>
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--shell-group-title)]">{t.pages.openplat.sandboxLabel}</span>
          <span className="text-[var(--shell-content-text)]">{app.sandbox ? t.pages.openplat.sandboxOn : '—'}</span>
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-[var(--shell-group-title)]">{t.pages.openplat.statusLabel}</span>
          <span className={`inline-flex h-5 items-center rounded-full px-2 text-[11px] font-medium ${appStatusClass}`}>{appStatusLabel}</span>
        </div>
      </div>

      <TabBar
        tabs={[
          { key: 'subs', label: `${t.pages.openplat.subsTab} (${subs.length})` },
          { key: 'deliveries', label: `${t.pages.openplat.dlvTab} (${appDeliveries.length})` },
        ]}
        value={tab}
        onChange={setTab}
      />

      {tab === 'subs' && (
        subs.length === 0 ? <EmptyState text={t.pages.openplat.noSubs} /> : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="h-9 px-2">{t.pages.openplat.fEvents}</TableHead>
                <TableHead className="h-9 px-2">{t.pages.openplat.endpointLabel}</TableHead>
                <TableHead className="h-9 whitespace-nowrap px-2">{t.pages.openplat.createdLabel}</TableHead>
                <TableHead className="h-9 px-2 text-right">{t.pages.openplat.actionLabel}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {subs.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="h-10 whitespace-nowrap px-2 font-mono text-xs">{s.eventType}</TableCell>
                  <TableCell className="h-10 break-all px-2">{s.endpointUrl}</TableCell>
                  <TableCell className="h-10 whitespace-nowrap px-2">{formatTime(s.createdAt)}</TableCell>
                  <TableCell className="h-10 px-2 text-right">
                    <button className="cursor-pointer text-[var(--color-danger)] hover:underline disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => delSub(s.id)}>{t.pages.openplat.delSub}</button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      )}

      {tab === 'deliveries' && (
        appDeliveries.length === 0 ? <EmptyState text={t.pages.openplat.noDeliveries} /> : (
          <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="h-9 px-2">{t.pages.openplat.fEvents}</TableHead>
                <TableHead className="h-9 px-2">{t.pages.openplat.eventIdLabel}</TableHead>
                <TableHead className="h-9 whitespace-nowrap px-2">{t.pages.openplat.statusLabel}</TableHead>
                <TableHead className="h-9 px-2">{t.pages.openplat.responseLabel}</TableHead>
                <TableHead className="h-9 px-2 text-right">{t.pages.openplat.actionLabel}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {appDeliveries.slice(0, 20).map((d) => (
                <TableRow key={d.id}>
                  <TableCell className="h-10 whitespace-nowrap px-2 font-mono text-xs">{d.eventType}</TableCell>
                  <TableCell className="h-10 break-all px-2 font-mono text-xs">{d.eventId}</TableCell>
                  <TableCell className="h-10 whitespace-nowrap px-2">{statusLabel(d.status)} · {d.attempts}</TableCell>
                  <TableCell className="h-10 px-2">{d.lastError || (d.httpStatus ? `HTTP ${d.httpStatus}` : '')}</TableCell>
                  <TableCell className="h-10 whitespace-nowrap px-2 text-right">
                    {d.status !== 1
                      ? <button className="cursor-pointer text-[var(--color-text-link)] hover:underline disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={() => requeue(d.id)}>{t.pages.openplat.requeue}</button>
                      : <span className="text-[var(--shell-group-title)]">—</span>}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {appDeliveries.length > 20 && (
            <div className="pt-2 text-xs text-[var(--shell-group-title)]">{t.pages.openplat.dlvTruncated}</div>
          )}
          </>
        )
      )}

      {error && <div className="mt-3"><ErrorBanner message={error} /></div>}

      {subOpen && (
        <SubForm appId={app.id} onClose={closeSubForm}
          onCreated={() => { closeSubForm(); load() }} />
      )}
    </Drawer>
  )
}