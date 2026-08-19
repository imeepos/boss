// 子公司新建/编辑抽屉:原型表单的编码/名称两必填项(后端契约仅收 code/name)。
// 契约: POST /legal-entities、PUT /legal-entities/{id}(menu:company 权限)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { useT } from '../../../i18n'
import type { LegalEntityRow } from './filter'

export function CompanyForm({
  initial, onDone, onCancel,
}: {
  initial: LegalEntityRow | null
  onDone: () => void
  onCancel: () => void
}) {
  const t = useT().pages.company
  const [code, setCode] = useState(initial?.code ?? '')
  const [name, setName] = useState(initial?.name ?? '')
  const [err, setErr] = useState('')

  const save = async () => {
    if (!code.trim() || !name.trim()) {
      setErr(t.requiredHint)
      return
    }
    const body = { code: code.trim(), name: name.trim() }
    const path = initial ? `/legal-entities/${initial.id}` : '/legal-entities'
    try {
      await apiFetch(path, { method: initial ? 'PUT' : 'POST', body })
      onDone()
    } catch {
      setErr(t.saveFail)
    }
  }

  return (
    <Drawer
      title={initial ? `${t.edit} · ${initial.code}` : t.add}
      onClose={onCancel}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onCancel}>{t.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={save}>{t.save}</button>
        </>
      }
    >
      {err && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" role="alert">{err}</div>}
      <div className="flex flex-col gap-3.5">
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.codeLabel}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={code} placeholder={t.codePlaceholder}
            onChange={(e) => setCode(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.nameLabel}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={name} placeholder={t.namePlaceholder}
            onChange={(e) => setName(e.target.value)} />
        </div>
      </div>
    </Drawer>
  )
}
