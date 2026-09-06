// 编辑抽屉:契约 PUT /assets/:assetId {type?, modelId?, tagId?, batchId?}(仅传改动字段)。
// 状态与部署地址只读展示;批次仅 IN_STOCK 态开放,其余状态禁用并提示走业务流转。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { useT } from '../../../i18n'
import type { AssetBatchRow, AssetModelRow, AssetRow, TagRow } from '../types'
import { AssetFormFields } from './AssetFormFields'
import { buildEditPayload, formErrOf, modelLabel, type AssetFormState, type FormErr } from './logic'

const label = 'text-xs text-[var(--shell-group-title)]'
const value = 'text-[13px] text-[var(--shell-content-text)]'

export function EditDrawer({ asset, onClose, onSaved }: {
  asset: AssetRow
  onClose: () => void
  onSaved: () => void
}) {
  const t = useT()
  const a = t.pages.assetPage
  const [form, setForm] = useState<AssetFormState>({ batchId: asset.batchId, modelId: asset.modelId, type: asset.type, tagId: asset.tagId })
  const [batches, setBatches] = useState<AssetBatchRow[]>([])
  const [models, setModels] = useState<AssetModelRow[]>([])
  const [tags, setTags] = useState<TagRow[]>([])
  const [err, setErr] = useState<FormErr>('')
  const [apiError, setApiError] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    apiFetch<{ items: AssetBatchRow[] }>('/assets/batches').then((d) => setBatches(d?.items ?? [])).catch(() => setBatches([]))
    // 型号下拉=启用项 + 当前挂型号(停用后仍需回显)
    apiFetch<{ items: AssetModelRow[] }>('/asset-models').then((d) => setModels((d?.items ?? []).filter((m) => m.isActive || m.id === asset.modelId))).catch(() => setModels([]))
    // 标签下拉=UNBOUND + 当前绑定标签(换绑/解绑都从当前态出发)
    apiFetch<{ items: TagRow[] }>('/tags').then((d) => setTags((d?.items ?? []).filter((x) => x.status === 'UNBOUND' || x.tagId === asset.tagId))).catch(() => setTags([]))
  }, [asset.assetId, asset.modelId, asset.tagId]) // eslint-disable-line react-hooks/exhaustive-deps

  const batchLocked = asset.status !== 'IN_STOCK'
  const modelOf = (id: number) => models.find((m) => m.id === id)
  const submit = async () => {
    if (busy) return
    const e = formErrOf(form)
    setErr(e)
    if (e) return
    setBusy(true)
    setApiError('')
    try {
      await apiFetch('/assets/' + String(asset.assetId), { method: 'PUT', body: buildEditPayload(form, asset) })
      toast.success(a.editOk)
      onSaved()
      onClose()
    } catch (err2) {
      setApiError(err2 instanceof Error ? err2.message : String(err2))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Drawer title={a.editTitle + ' · ' + asset.assetCode} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={submit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="mb-3.5 grid grid-cols-2 gap-3.5">
        <div className="flex flex-col gap-1.5">
          <span className={label}>{a.dStatus}</span>
          <span className={value}><StatusTag domain="asset" value={asset.status} /></span>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className={label}>{a.dAddress}</span>
          <span className={value}>{asset.addressId ? '#' + String(asset.addressId) : '—'}</span>
        </div>
      </div>
      <AssetFormFields form={form}
        batchOptions={batches.map((b) => ({ value: String(b.id), label: '#' + String(b.id) + ' ' + [b.code, b.name].filter(Boolean).join(' ') }))}
        modelOptions={models.map((m) => ({ value: String(m.id), label: modelLabel(m) }))}
        tagOptions={tags.map((x) => ({ value: String(x.tagId), label: (x.tagId === asset.tagId && x.status !== 'UNBOUND' ? '● ' : '') + x.tagNo }))}
        typeValue={form.modelId ? (modelOf(form.modelId)?.category ?? asset.type) : form.type}
        typeReadonly={!!form.modelId}
        batchDisabled={batchLocked} batchHint={batchLocked ? a.editBatchLocked : ''} error={err}
        onBatch={(batchId) => { setForm((f) => ({ ...f, batchId })) }}
        onModel={(modelId) => { setForm((f) => ({ ...f, modelId })) }}
        onType={(type) => { setForm((f) => ({ ...f, type })) }}
        onTag={(tagId) => { setForm((f) => ({ ...f, tagId })) }} />
      {apiError && <div className="mt-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{apiError}</div>}
    </Drawer>
  )
}