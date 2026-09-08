// 型号字典抽屉:契约 GET /asset-models(含停用)、POST /asset-models、PUT /asset-models/{id}、
// POST /asset-models/{id}/disable|enable。建档/编辑下拉数据源与既有端点一致(GET /asset-models)。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useConfirm } from '../../../components/ConfirmDialog'
import { Drawer } from '../../../components/Drawer'
import { ActionLink, ActionLinks, ActionSep, TableStateRow } from '../../../components/business'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton } from '../../../components/business/submit-button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import type { AssetModelRow } from '../types'
import { buildModelPayload, emptyModelForm, modelFormErr, type ModelFormState } from './dictLogic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export function ModelDictDrawer({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const t = useT()
  const a = t.pages.assetPage
  const confirm = useConfirm()
  const [rows, setRows] = useState<AssetModelRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [editing, setEditing] = useState<AssetModelRow | null>(null)
  const [form, setForm] = useState<ModelFormState>(emptyModelForm)
  const [formErr, setFormErr] = useState('')
  const [loaded, setLoaded] = useState(false)

  const load = useCallback(() => {
    setError('')
    apiFetch<{ items: AssetModelRow[] }>('/asset-models')
      .then((x) => { setRows(x?.items ?? []); setLoaded(true) })
      .catch((e) => { setError(e instanceof Error ? e.message : a.loadFail); setLoaded(true) })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(load, [load])

  const startEdit = (m: AssetModelRow) => {
    setEditing(m)
    setForm({ vendor: m.vendor, model: m.model, category: m.category, partNumber: m.partNumber })
    setFormErr('')
  }
  const resetForm = () => { setEditing(null); setForm(emptyModelForm); setFormErr('') }

  const submit = async () => {
    if (busy) return
    const key = modelFormErr(form)
    setFormErr(key)
    if (key) return
    setBusy(true)
    try {
      await apiFetch(editing ? '/asset-models/' + editing.id : '/asset-models', {
        method: editing ? 'PUT' : 'POST',
        body: buildModelPayload(form),
      })
      toast.success(a.modelSaveOk)
      resetForm()
      load()
      onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const toggle = async (m: AssetModelRow) => {
    const enabling = !m.isActive
    if (!enabling) {
      const name = [m.vendor, m.model].filter(Boolean).join(' ')
      const ok = await confirm(a.modelDisableConfirm.replace('{name}', name), { danger: true, title: a.modelDisabled })
      if (!ok) return
    }
    setBusy(true)
    try {
      await apiFetch('/asset-models/' + m.id + (enabling ? '/enable' : '/disable'), { method: 'POST' })
      toast.success(enabling ? a.modelEnabled : a.modelDisabled)
      load()
      onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : a.loadFail)
    } finally {
      setBusy(false)
    }
  }

  const submitState = busy ? 'loading' : (error ? 'failed' : 'idle')
  const submitLabels = { idle: editing ? t.pages.company.save : a.modelCreate, loading: t.pages.account.submitting, success: editing ? t.pages.company.save : a.modelCreate, failed: a.loadFail }
  const cols = a.modelCols
  return (
    <Drawer title={a.modelsTitle} onClose={onClose} width={720}
      footer={
        <>
          <ToolbarButton onClick={onClose} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
          <SubmitButton state={submitState} labels={submitLabels} disabled={busy} onClick={submit} />
        </>
      }>
      <div className="flex flex-col gap-3">
        <div className="rounded-sm border border-[var(--shell-side-border)] p-3">
          <div className="mb-2 text-[13px] font-medium text-[var(--shell-heading)]">{editing ? a.modelEdit : a.modelCreate}</div>
          <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
            <FormField label={a.modelVendor}><input className={input} placeholder={a.modelVendor} value={form.vendor} onChange={(e) => setForm({ ...form, vendor: e.target.value })} /></FormField>
            <FormField label={a.modelName}><input className={input} placeholder={a.modelName} value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })} /></FormField>
            <FormField label={a.modelCategory}><input className={input} placeholder={a.modelCategory} value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} /></FormField>
            <FormField label={a.modelPart}><input className={input} placeholder={a.modelPart} value={form.partNumber} onChange={(e) => setForm({ ...form, partNumber: e.target.value })} /></FormField>
          </div>
          <div className="mt-2 flex items-center gap-2">
            {editing && <ToolbarButton onClick={resetForm} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>}
          </div>
          {formErr && <ErrorBanner message={a[formErr as 'eModelRequired']} />}
        </div>
        {error && <ErrorBanner message={error} />}
        <Table>
          <TableHeader>
            <TableRow>{cols.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((m) => (
              <TableRow key={m.id}>
                <TableCell>{m.vendor || '—'}</TableCell>
                <TableCell>{m.model}</TableCell>
                <TableCell>{m.category}</TableCell>
                <TableCell>{m.partNumber || '—'}</TableCell>
                <TableCell>
                  <span className={m.isActive ? 'text-[var(--color-success)]' : 'text-[var(--shell-group-title)]'}>
                    {m.isActive ? a.modelEnabled : a.modelDisabled}
                  </span>
                </TableCell>
                <TableCell>
                  <ActionLinks>
                    <ActionLink onClick={() => startEdit(m)} label={t.pages.assetPage.edit} testId={`model-edit-${m.id}`} />
                    <ActionSep />
                    {m.isActive
                      ? <ActionLink onClick={() => toggle(m)} label={a.modelDisable} testId={`model-disable-${m.id}`} />
                      : <ActionLink onClick={() => toggle(m)} label={a.modelEnable} testId={`model-enable-${m.id}`} />}
                  </ActionLinks>
                </TableCell>
              </TableRow>
            ))}
            {!rows.length && <TableStateRow colSpan={cols.length} loading={!loaded} text={a.empty} />}
          </TableBody>
        </Table>
      </div>
    </Drawer>
  )
}
