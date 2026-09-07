// 新增标签表单字段区(展示组件):label/必填/错误提示走 FormField;载荷组装见 ./logic。
import { useT } from '../../../i18n'
import { FormField } from '../../../components/business/form-field'
import type { TagFormState } from './logic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export interface TagFormFieldsProps {
  form: TagFormState
  error: string
  onTagNo: (v: string) => void
  onEpc: (v: string) => void
  onBand: (v: string) => void
}

export function TagFormFields({ form, error, onTagNo, onEpc, onBand }: TagFormFieldsProps) {
  const t = useT()
  const g = t.pages.tagPage
  return (
    <div className="flex flex-col gap-3.5">
      <FormField label={g.fTagNo} required error={error === 'eTagNo' ? g.eTagNo : undefined}>
        <input className={input} value={form.tagNo} placeholder={g.fTagNo} onChange={(e) => onTagNo(e.target.value)} />
      </FormField>
      <FormField label={g.fEpc} required>
        <input className={input} value={form.epcCode} placeholder={g.fEpc} onChange={(e) => onEpc(e.target.value)} />
      </FormField>
      <FormField label={g.fBand}>
        <input className={input} value={form.band} placeholder={g.pBand} onChange={(e) => onBand(e.target.value)} />
      </FormField>
    </div>
  )
}
