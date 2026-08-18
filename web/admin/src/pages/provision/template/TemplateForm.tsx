// 模板新建抽屉:契约 POST /provision-templates(body: legalEntityId/code/name)。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
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
          <button className="org-btn" onClick={onCancel}>{p.cancel}</button>
          <button className="org-btn org-btn-primary" disabled={busy} onClick={save}>{p.save}</button>
        </>
      }
    >
      {err && <div className="org-error" role="alert">{err}</div>}
      <div className="org-form">
        <div className="org-field">
          <label><span className="req">*</span>{p.legalEntity}</label>
          <select className="org-select" value={legalEntityId ? String(legalEntityId) : ''}
            onChange={(e) => setLegalEntityId(Number(e.target.value) || 0)}>
            <option value="">{p.legalEntityPlaceholder}</option>
            {entities.map((x) => <option key={x.id} value={x.id}>{x.name} ({x.code})</option>)}
          </select>
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{p.codeLabel}</label>
          <input className="org-input" value={code} placeholder={p.codePlaceholder}
            onChange={(e) => setCode(e.target.value)} />
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{p.nameLabel}</label>
          <input className="org-input" value={name} placeholder={p.namePlaceholder}
            onChange={(e) => setName(e.target.value)} />
        </div>
      </div>
    </Drawer>
  )
}
