// 批次管理抽屉:契约 GET /assets/batches(既有列表端点)+ POST /asset-batches 新建。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { TableStateRow } from '../../../components/business'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
import type { AssetBatchRow } from '../types'
import { batchFormErr, buildBatchPayload, emptyBatchForm, type BatchFormState } from './dictLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function BatchDrawer({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const t = useT()
  const a = t.pages.assetPage
  const [rows, setRows] = useState<AssetBatchRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [form, setForm] = useState<BatchFormState>(emptyBatchForm)
  const [formErr, setFormErr] = useState('')
  const [loaded, setLoaded] = useState(false)

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: AssetBatchRow[] }>('/assets/batches')
      .then((x) => { setRows(x?.items ?? []); setLoaded(true) })
      .catch((e) => { setError(e instanceof Error ? e.message : a.loadFail); setLoaded(true) })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(load, [load])

  const submit = async () => {
    if (busy) return
    const key = batchFormErr(form)
    setFormErr(key)
    if (key) return
    setBusy(true)
    try {
      await apiFetch('/asset-batches', { method: 'POST', body: buildBatchPayload(form) })
      toast.success(a.batchSaveOk)
      setForm(emptyBatchForm)
      load()
      onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const cols = a.batchCols.slice(0, 2)
  return (
    <Drawer title={a.batchesTitle} onClose={onClose} width={560}
      footer={<button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      <div className="flex flex-col gap-3">
        <div className="rounded-sm border border-[var(--shell-side-border)] p-3">
          <div className="mb-2 text-[13px] font-medium text-[var(--shell-heading)]">{a.batchCreate}</div>
          <div className="flex flex-wrap items-center gap-2">
            <FormField label={a.batchCols[0]}><input className={input} placeholder={a.batchCols[0]} value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} /></FormField>
            <FormField label={a.batchCols[1]}><input className={input} placeholder={a.batchCols[1]} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></FormField>
            <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)] disabled:cursor-not-allowed disabled:opacity-50" disabled={busy} onClick={submit}>{a.batchCreate}</button>
          </div>
          {formErr && <ErrorBanner message={a[formErr as 'eBatchRequired']} />}
        </div>
        {error && <ErrorBanner message={error} />}
        <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr>{cols.map((x) => <th key={x} className="h-9 px-2 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr>
              </thead>
              <tbody>
                {rows.map((b) => (
                  <tr key={b.id}>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] font-mono text-[var(--shell-content-text)]">{b.code || '#' + b.id}</td>
                    <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]">{b.name || '—'}</td>
                  </tr>
                ))}
                {!rows.length && <TableStateRow colSpan={cols.length} loading={!loaded} text={a.empty} />}
              </tbody>
            </table>
          </div>
      </div>
    </Drawer>
  )
}
