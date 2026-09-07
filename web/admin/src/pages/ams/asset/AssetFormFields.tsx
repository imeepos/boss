// 建档/编辑共用表单字段区:批次/型号/类型/标签四控件(纯展示,状态由抽屉持有)。
// 下拉一律 Dropdown 组件;类型=白名单下拉(P4-T2,选中型号后禁用由型号类别派生);
// 批次可由调用方禁用(非 IN_STOCK 态)。
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { useT } from '../../../i18n'
import { ASSET_TYPES } from '../types'
import type { FormErr } from './logic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)] disabled:cursor-not-allowed disabled:bg-[var(--shell-input-disabled-bg)] disabled:text-[var(--shell-input-placeholder)]'

export interface AssetFormFieldsProps {
  form: { batchId: number; modelId: number; type: string; tagId: number; sn: string; mac: string; loid: string }
  batchOptions: { value: string; label: string }[]
  modelOptions: { value: string; label: string }[]
  tagOptions: { value: string; label: string }[]
  typeValue: string
  typeReadonly: boolean
  batchDisabled: boolean
  batchHint: string
  error: FormErr
  onBatch: (id: number) => void
  onModel: (id: number) => void
  onType: (v: string) => void
  onTag: (id: number) => void
  onSn: (v: string) => void
  onMac: (v: string) => void
  onLoid: (v: string) => void
}

export function AssetFormFields(p: AssetFormFieldsProps) {
  const t = useT()
  const a = t.pages.assetPage
  return (
    <div className="flex flex-col gap-3.5">
      <FormField label={a.fBatch} required error={p.error === 'batch' ? a.eBatch : undefined} hint={p.batchHint || undefined}>
        <Dropdown value={p.form.batchId ? String(p.form.batchId) : ''} options={p.batchOptions}
          onChange={(v) => p.onBatch(Number(v) || 0)} ariaLabel={a.fBatch} placeholder={a.pBatch} disabled={p.batchDisabled} />
      </FormField>
      <FormField label={a.fModel}>
        <Dropdown value={p.form.modelId ? String(p.form.modelId) : ''} options={p.modelOptions}
          onChange={(v) => p.onModel(Number(v) || 0)} ariaLabel={a.fModel} placeholder={a.pModel} />
      </FormField>
      <FormField label={a.fType} error={p.error === 'type' ? a.eType : undefined}>
        <Dropdown value={p.typeValue} ariaLabel={a.fType} placeholder={a.pType}
          disabled={p.typeReadonly}
          options={ASSET_TYPES.map((v) => ({ value: v, label: v }))}
          onChange={(v) => p.onType(v)} />
      </FormField>
      <FormField label={a.fIdentitySn}>
        <input className={input} value={p.form.sn} placeholder={a.pIdentitySn}
          onChange={(e) => p.onSn(e.target.value)} />
      </FormField>
      <FormField label={a.fIdentityMac}>
        <input className={input} value={p.form.mac} placeholder={a.pIdentityMac}
          onChange={(e) => p.onMac(e.target.value)} />
      </FormField>
      <FormField label={a.fIdentityLoid}>
        <input className={input} value={p.form.loid} placeholder={a.pIdentityLoid}
          onChange={(e) => p.onLoid(e.target.value)} />
      </FormField>
      <FormField label={a.fTag}>
        <Dropdown value={p.form.tagId ? String(p.form.tagId) : ''} options={p.tagOptions}
          onChange={(v) => p.onTag(Number(v) || 0)} ariaLabel={a.fTag} placeholder={a.pTag} />
      </FormField>
    </div>
  )
}