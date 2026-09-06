// 新增标签表单字段区(展示组件):必填标记与错误提示;载荷组装见 ./logic。
import { useT } from '../../../i18n'
import type { TagFormState } from './logic'

const input = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const errBanner = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

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
      <div className="flex flex-col gap-1.5">
        <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{g.fTagNo}</label>
        <input className={input} value={form.tagNo} placeholder={g.fTagNo} onChange={(e) => onTagNo(e.target.value)} />
      </div>
      <div className="flex flex-col gap-1.5">
        <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{g.fEpc}</label>
        <input className={input} value={form.epcCode} placeholder={g.fEpc} onChange={(e) => onEpc(e.target.value)} />
      </div>
      <div className="flex flex-col gap-1.5">
        <label>{g.fBand}</label>
        <input className={input} value={form.band} placeholder={g.pBand} onChange={(e) => onBand(e.target.value)} />
      </div>
      {error && <div className={errBanner}>{g[error as 'eTagNo']}</div>}
    </div>
  )
}
