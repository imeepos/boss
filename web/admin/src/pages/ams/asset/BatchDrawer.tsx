// 批次管理抽屉:契约 GET /assets/batches(既有列表端点)+ POST /asset-batches 新建。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { TableStateRow } from '../../../components/business'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
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

  const submitState = busy ? 'loading' : (error ? 'failed' : 'idle')
  const submitLabels = { idle: a.batchCreate, loading: t.pages.account.submitting, success: a.batchCreate, failed: a.loadFail }
  const cols = a.batchCols.slice(0, 2)
  return (
    <Drawer title={a.batchesTitle} onClose={onClose} width={560}
      footer={
        <>
          <ToolbarButton onClick={onClose} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
          <SubmitButton state={submitState} labels={submitLabels} disabled={busy} onClick={submit} />
        </>
      }>
      <div className="flex flex-col gap-3">
        <div className="rounded-sm border border-[var(--shell-side-border)] p-3">
          <div className="mb-2 text-[13px] font-medium text-[var(--shell-heading)]">{a.batchCreate}</div>
          <div className="flex flex-wrap items-center gap-2">
            <FormField label={a.batchCols[0]}><input className={input} placeholder={a.batchCols[0]} value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} /></FormField>
            <FormField label={a.batchCols[1]}><input className={input} placeholder={a.batchCols[1]} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></FormField>
          </div>
          {formErr && <ErrorBanner message={a[formErr as 'eBatchRequired']} />}
        </div>
        {error && <ErrorBanner message={error} />}
        <Table>
          <TableHeader>
            <TableRow>{cols.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((b) => (
              <TableRow key={b.id}>
                <TableCell className="font-mono">{b.code || '#' + b.id}</TableCell>
                <TableCell>{b.name || '—'}</TableCell>
              </TableRow>
            ))}
            {!rows.length && <TableStateRow colSpan={cols.length} loading={!loaded} text={a.empty} />}
          </TableBody>
        </Table>
      </div>
    </Drawer>
  )
}
