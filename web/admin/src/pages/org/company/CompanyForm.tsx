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
          <button className="org-btn" onClick={onCancel}>{t.cancel}</button>
          <button className="org-btn org-btn-primary" onClick={save}>{t.save}</button>
        </>
      }
    >
      {err && <div className="org-error" role="alert">{err}</div>}
      <div className="org-form">
        <div className="org-field">
          <label><span className="req">*</span>{t.codeLabel}</label>
          <input className="org-input" value={code} placeholder={t.codePlaceholder}
            onChange={(e) => setCode(e.target.value)} />
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{t.nameLabel}</label>
          <input className="org-input" value={name} placeholder={t.namePlaceholder}
            onChange={(e) => setName(e.target.value)} />
        </div>
      </div>
    </Drawer>
  )
}
