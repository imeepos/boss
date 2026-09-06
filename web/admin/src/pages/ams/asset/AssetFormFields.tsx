// 建档/编辑共用表单字段区:批次/型号/类型/标签四控件(纯展示,状态由抽屉持有)。
// 下拉一律 Dropdown 组件;类型选中型号后只读;批次可由调用方禁用(非 IN_STOCK 态)。
import { Dropdown } from '../../../components/Dropdown'
import { useT } from '../../../i18n'
import type { FormErr } from './logic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)] disabled:cursor-not-allowed disabled:bg-[var(--shell-input-disabled-bg)] disabled:text-[var(--shell-input-placeholder)]'

function Field({ label, required, children }: { label: string; required?: boolean; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-[13px] text-[var(--shell-content-text)]">{required && <span className="mr-0.5 text-[var(--color-danger)]">*</span>}{label}</label>
      {children}
    </div>
  )
}

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
      <Field label={a.fBatch} required>
        <Dropdown value={p.form.batchId ? String(p.form.batchId) : ''} options={p.batchOptions}
          onChange={(v) => p.onBatch(Number(v) || 0)} ariaLabel={a.fBatch} placeholder={a.pBatch} disabled={p.batchDisabled} />
      </Field>
      <Field label={a.fModel}>
        <Dropdown value={p.form.modelId ? String(p.form.modelId) : ''} options={p.modelOptions}
          onChange={(v) => p.onModel(Number(v) || 0)} ariaLabel={a.fModel} placeholder={a.pModel} />
      </Field>
      <Field label={a.fType}>
        <input className={input} value={p.typeValue} placeholder={a.pType} readOnly={p.typeReadonly}
          onChange={(e) => p.onType(e.target.value)} />
      </Field>
      <Field label={a.fSn}>
        <input className={input} value={p.form.sn} placeholder={a.pSn}
          onChange={(e) => p.onSn(e.target.value)} />
      </Field>
      <Field label={a.fMac}>
        <input className={input} value={p.form.mac} placeholder={a.pMac}
          onChange={(e) => p.onMac(e.target.value)} />
      </Field>
      <Field label={a.fLoid}>
        <input className={input} value={p.form.loid} placeholder={a.pLoid}
          onChange={(e) => p.onLoid(e.target.value)} />
      </Field>
      <Field label={a.fTag}>
        <Dropdown value={p.form.tagId ? String(p.form.tagId) : ''} options={p.tagOptions}
          onChange={(v) => p.onTag(Number(v) || 0)} ariaLabel={a.fTag} placeholder={a.pTag} />
      </Field>
      {p.batchHint && <span className="text-[11px] text-[var(--shell-group-title)]">{p.batchHint}</span>}
      {p.error === 'batch' && <span className="text-[11px] text-[var(--color-danger)]">{a.eBatch}</span>}
      {p.error === 'type' && <span className="text-[11px] text-[var(--color-danger)]">{a.eType}</span>}
    </div>
  )
}