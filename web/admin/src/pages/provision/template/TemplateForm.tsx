// 模板新建抽屉:契约 POST /provision-templates(body: legalEntityId/code/name)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { useT } from '../../../i18n'
import type { LegalEntityRow } from '../../org/company/filter'

export function TemplateForm({
  entities, onDone, onCancel,
}: {
  entities: LegalEntityRow[]
  onDone: () => void
  onCancel: () => void
}) {
  const p = useT().pages.templatePage
  const [legalEntityId, setLegalEntityId] = useState(entities[0]?.id ?? 0)
  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)

  const save = async () => {
    if (!legalEntityId || !code.trim() || !name.trim()) {
      setErr(p.requiredHint)
      return
    }
    setBusy(true)
    try {
      await apiFetch('/provision-templates', {
        method: 'POST',
        body: { legalEntityId, code: code.trim(), name: name.trim() },
      })
      onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : p.saveFail)
      setBusy(false)
    }
  }

  return (
    <Drawer
      title={p.create}
      onClose={onCancel}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onCancel}>{p.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={save}>{p.save}</button>
        </>
      }
    >
      {err && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" role="alert">{err}</div>}
      <div className="flex flex-col gap-3.5">
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{p.legalEntity}</label>
          <Dropdown
            value={legalEntityId ? String(legalEntityId) : ''}
            options={[{ value: '', label: p.legalEntityPlaceholder }, ...entities.map((x) => ({ value: String(x.id), label: `${x.name} (${x.code})` }))]}
            onChange={(v) => setLegalEntityId(Number(v) || 0)}
            ariaLabel={p.legalEntityPlaceholder}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{p.codeLabel}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={code} placeholder={p.codePlaceholder}
            onChange={(e) => setCode(e.target.value)} />
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{p.nameLabel}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={name} placeholder={p.namePlaceholder}
            onChange={(e) => setName(e.target.value)} />
        </div>
      </div>
    </Drawer>
  )
}
