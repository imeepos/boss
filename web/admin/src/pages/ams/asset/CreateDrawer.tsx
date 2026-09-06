// 建档抽屉:契约 POST /assets {batchId, modelId?, type?, tagId?}。
// 数据源:GET /assets/batches、/asset-models(仅 isActive 项)、/tags(仅 UNBOUND)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import type { AssetBatchRow, AssetModelRow, TagRow } from '../types'
import { AssetFormFields } from './AssetFormFields'
import { buildCreatePayload, emptyForm, formErrOf, modelLabel, typeValue, type AssetFormState, type FormErr } from './logic'

export function CreateDrawer({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const t = useT()
  const a = t.pages.assetPage
  const [form, setForm] = useState<AssetFormState>(emptyForm)
  const [batches, setBatches] = useState<AssetBatchRow[]>([])
  const [models, setModels] = useState<AssetModelRow[]>([])
  const [tags, setTags] = useState<TagRow[]>([])
  const [err, setErr] = useState<FormErr>('')
  const [apiError, setApiError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    apiFetch<{ items: AssetBatchRow[] }>('/assets/batches').then((d) => setBatches(d?.items ?? [])).catch(() => setBatches([]))
    apiFetch<{ items: AssetModelRow[] }>('/asset-models').then((d) => setModels((d?.items ?? []).filter((m) => m.isActive))).catch(() => setModels([]))
    apiFetch<{ items: TagRow[] }>('/tags').then((d) => setTags((d?.items ?? []).filter((x) => x.status === 'UNBOUND'))).catch(() => setTags([]))
  }, [])

  const modelOf = (id: number) => models.find((m) => m.id === id)
  const submit = async () => {
    if (busy) return
    const e = formErrOf(form)
    setErr(e)
    if (e) return
    setBusy(true)
    setApiError('')
    try {
      await apiFetch('/assets', { method: 'POST', body: buildCreatePayload(form) })
      toast.success(a.createOk)
      onSaved()
      onClose()
    } catch (err2) {
      setApiError(err2 instanceof Error ? err2.message : String(err2))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer title={a.createTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <AssetFormFields form={form}
        batchOptions={batches.map((b) => ({ value: String(b.id), label: '#' + String(b.id) + ' ' + [b.code, b.name].filter(Boolean).join(' ') }))}
        modelOptions={models.map((m) => ({ value: String(m.id), label: modelLabel(m) }))}
        tagOptions={tags.map((x) => ({ value: String(x.tagId), label: x.tagNo }))}
        typeValue={typeValue(form, modelOf)} typeReadonly={!!form.modelId}
        batchDisabled={false} batchHint="" error={err}
        onBatch={(batchId) => { setForm((f) => ({ ...f, batchId })) }}
        onModel={(modelId) => { setForm((f) => ({ ...f, modelId })) }}
        onType={(type) => { setForm((f) => ({ ...f, type })) }}
        onTag={(tagId) => { setForm((f) => ({ ...f, tagId })) }}
        onSn={(sn) => { setForm((f) => ({ ...f, sn })) }}
        onMac={(mac) => { setForm((f) => ({ ...f, mac })) }}
        onLoid={(loid) => { setForm((f) => ({ ...f, loid })) }} />
      {apiError && <div className="mt-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{apiError}</div>}
    </Drawer>
  )
}