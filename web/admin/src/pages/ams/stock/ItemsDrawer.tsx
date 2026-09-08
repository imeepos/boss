// 盘点差异明细抽屉:扫码回填 + 逐条处置(CONFIRM/FIX/ESCALATE)。
// 契约 GET /stocktakes/:taskId/items、POST .../scans、POST .../items/:itemId/handle。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { statusTagLabel } from '../../../components/StatusTag'
import { ActionLink, ActionLinks, ActionSep, TableStateRow } from '../../../components/business'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { SCAN_STATUSES, type AssetRow, type StocktakeItemRow } from '../types'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

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
      toast.success(s.handleOk)
      setNote('')
      load()
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : s.actionFail)
      setBusy(false)
    }
  }

  const kindText = (k: string) => (s as unknown as Record<string, string>)[`kind${k}`] ?? k
  const resText = (r: string) => (s as unknown as Record<string, string>)[`res${r}`] ?? r

  const scanState = busy ? 'loading' : (error ? 'failed' : 'idle')
  const scanLabels = { idle: s.scan, loading: t.pages.account.submitting, success: s.scanOk, failed: s.scanFail }

  return (
    <Drawer title={`${s.detail} · #${taskId}`} onClose={onClose} width={860}>
      <div className="flex flex-col gap-3">
        {canEdit && (
          <div className="flex flex-wrap items-end gap-2 rounded-sm border border-[var(--shell-side-border)] p-3">
            <FormField label={s.fAsset} required>
              <SimplePicker value={assetId} onChange={setAssetId} ariaLabel={s.fAsset}
                placeholder={s.pAssetId}
                search={async (kw) => {
                  const d = await apiFetch<{ items: AssetRow[] }>('/assets', { query: { q: kw || undefined, limit: 20 } })
                  return (d?.items ?? []).map((a) => ({ value: String(a.assetId), label: (a.assetCode || '#' + a.assetId) + ' · ' + a.status }))
                }}
                errorText={s.loadFail}
                minWidth={280}
              />
            </FormField>
            <FormField label={s.fScanStatus}>
              <Dropdown value={scanStatus} triggerStyle={{ width: 144 }}
                options={SCAN_STATUSES.map((x) => ({ value: x, label: statusTagLabel('asset', x, t.common.statusTags) }))}
                onChange={setScanStatus} ariaLabel={s.fScanStatus} />
            </FormField>
            <FormField label={s.notePrompt}>
              <input className={`${input} w-56`} value={note} onChange={(e) => setNote(e.target.value)} />
            </FormField>
            <SubmitButton state={scanState} labels={scanLabels} disabled={busy || !assetId} onClick={scan} />
          </div>
        )}
        <Table>
          <TableHeader>
            <TableRow>{s.itemCols.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
          </TableHeader>
          <TableBody>
            {items.map((it) => (
              <TableRow key={it.id}>
                <TableCell className="font-mono">#{it.id}</TableCell>
                <TableCell>{it.assetId}</TableCell>
                <TableCell>{it.expectedStatus || '—'}</TableCell>
                <TableCell>{it.scannedStatus || '—'}</TableCell>
                <TableCell>{kindText(it.kind)}</TableCell>
                <TableCell>{resText(it.resolution)}{it.handledBy > 0 ? ` · ${s.handledBy.replace('{id}', String(it.handledBy))}` : ''}</TableCell>
                <TableCell className="max-w-40 truncate" title={it.note}>{it.note || '—'}</TableCell>
                {canEdit && (
                  <TableCell>
                    {it.kind !== 'OK' && it.resolution === 'OPEN' && (
                      <ActionLinks>
                        <ActionLink onClick={() => handle(it, 'CONFIRM')} label={s.actConfirm} testId={`item-confirm-${it.id}`} />
                        <ActionSep />
                        <ActionLink onClick={() => handle(it, 'FIX')} label={s.actFix} testId={`item-fix-${it.id}`} />
                        <ActionSep />
                        <ActionLink onClick={() => handle(it, 'ESCALATE')} label={s.actEscalate} testId={`item-escalate-${it.id}`} />
                      </ActionLinks>
                    )}
                  </TableCell>
                )}
              </TableRow>
            ))}
            {!items.length && <TableStateRow colSpan={canEdit ? 8 : 7} loading={busy} text={s.emptyItems} />}
          </TableBody>
        </Table>
        {error && <ErrorBanner message={error} />}
      </div>
    </Drawer>
  )
}
