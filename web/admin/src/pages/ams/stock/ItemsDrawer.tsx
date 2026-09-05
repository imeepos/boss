// 盘点差异明细抽屉:扫码回填 + 逐条处置(CONFIRM/FIX/ESCALATE)。
// 契约 GET /stocktakes/:taskId/items、POST .../scans、POST .../items/:itemId/handle。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SCAN_STATUSES, type StocktakeItemRow } from '../types'

const td = 'h-10 px-2.5 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'
const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const btn = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-xs text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50'

export function ItemsDrawer({ taskId, canEdit, onClose, onChanged }: {
  taskId: number
  canEdit: boolean // 任务 DOING 才允许扫码/处置
  onClose: () => void
  onChanged: () => void // 行数据变化后刷新任务列表(进度/差异项)
}) {
  const t = useT()
  const s = t.pages.stock
  const [items, setItems] = useState<StocktakeItemRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [assetId, setAssetId] = useState('')
  const [scanStatus, setScanStatus] = useState('IN_STOCK')
  const [note, setNote] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: StocktakeItemRow[] }>(`/stocktakes/${taskId}/items`)
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [taskId]) // eslint-disable-line react-hooks/exhaustive-deps

  const scan = async () => {
    if (busy || !assetId) return
    setBusy(true)
    setError('')
    try {
      await apiFetch(`/stocktakes/${taskId}/scans`, {
        method: 'POST',
        body: { assetId: Number(assetId), status: scanStatus },
      })
      toast.success(s.scanOk)
      setAssetId('')
      load()
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : s.scanFail)
    } finally {
      setBusy(false)
    }
  }

  const handle = async (item: StocktakeItemRow, action: 'CONFIRM' | 'FIX' | 'ESCALATE') => {
    if (busy) return
    if ((action === 'FIX' || action === 'ESCALATE') && note.trim() === '') {
      setError(s.noteRequired)
      return
    }
    setBusy(true)
    setError('')
    try {
      await apiFetch(`/stocktakes/${taskId}/items/${item.id}/handle`, {
        method: 'POST',
        body: { action, note: note.trim() },
      })
      setNote('')
      load()
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : s.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const kindText = (k: string) => (s as unknown as Record<string, string>)[`kind${k}`] ?? k
  const resText = (r: string) => (s as unknown as Record<string, string>)[`res${r}`] ?? r

  return (
    <Drawer title={`${s.detail} · #${taskId}`} onClose={onClose} width={860}>
      <div className="flex flex-col gap-3">
        {canEdit && (
          <div className="flex flex-wrap items-end gap-2 rounded-sm border border-[var(--shell-side-border)] p-3">
            <div className="flex flex-col gap-1">
              <label className="text-xs text-[var(--shell-group-title)]">{s.fAssetId}</label>
              <input className={`${input} w-32`} value={assetId} placeholder={s.pAssetId}
                onChange={(e) => setAssetId(e.target.value.replace(/\D/g, ''))} />
            </div>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-[var(--shell-group-title)]">{s.fScanStatus}</label>
              <Dropdown value={scanStatus} triggerStyle={{ width: 144 }}
                options={SCAN_STATUSES.map((x) => ({ value: x, label: x }))}
                onChange={setScanStatus} ariaLabel={s.fScanStatus} />
            </div>
            <button className={`h-8 border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:opacity-50`} disabled={busy || !assetId} onClick={scan}>{s.scan}</button>
          </div>
        )}
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-xs">
            <thead>
              <tr>{s.itemCols.map((x) => <th key={x} className="h-9 px-2.5 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr>
            </thead>
            <tbody>
              {items.map((it) => (
                <tr key={it.id}>
                  <td className={td}>#{it.id}</td>
                  <td className={td}>{it.assetId}</td>
                  <td className={td}>{it.expectedStatus || '—'}</td>
                  <td className={td}>{it.scannedStatus || '—'}</td>
                  <td className={td}>{kindText(it.kind)}</td>
                  <td className={td}>{resText(it.resolution)}{it.handledBy > 0 ? ` · ${s.handledBy.replace('{id}', String(it.handledBy))}` : ''}</td>
                  <td className={`${td} max-w-40 truncate`} title={it.note}>{it.note || '—'}</td>
                  {canEdit && (
                    <td className={td}>
                      {it.kind !== 'OK' && it.resolution === 'OPEN' && (
                        <span className="inline-flex gap-1.5">
                          <button className={btn} disabled={busy} onClick={() => handle(it, 'CONFIRM')}>{s.actConfirm}</button>
                          <button className={btn} disabled={busy} onClick={() => handle(it, 'FIX')}>{s.actFix}</button>
                          <button className={btn} disabled={busy} onClick={() => handle(it, 'ESCALATE')}>{s.actEscalate}</button>
                        </span>
                      )}
                    </td>
                  )}
                </tr>
              ))}
              {!items.length && <tr><td className={`${td} text-center`} colSpan={canEdit ? 8 : 7}>{busy ? t.common.loading : s.emptyItems}</td></tr>}
            </tbody>
          </table>
        </div>
        {canEdit && (
          <div className="flex flex-col gap-1">
            <label className="text-xs text-[var(--shell-group-title)]">{s.notePrompt}</label>
            <input className={input} value={note} onChange={(e) => setNote(e.target.value)} />
          </div>
        )}
        {error && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
      </div>
    </Drawer>
  )
}
