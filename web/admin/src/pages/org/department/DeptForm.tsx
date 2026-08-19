// 部门新建/编辑 Drawer 表单:挂靠子公司 + 名称;子公司内同名唯一(40900)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import type { DepartmentRow } from './filter'

export interface DeptFormValues {
  id: number
  legalEntityId: number
  name: string
}

export function emptyDeptForm(): DeptFormValues {
  return { id: 0, legalEntityId: 0, name: '' }
}

export function rowToDeptForm(r: DepartmentRow): DeptFormValues {
  return { id: r.id, legalEntityId: r.legalEntityId, name: r.name }
}

export function DeptFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: DeptFormValues
  onChange: (v: DeptFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const [entities, setEntities] = useState<{ id: number; code: string; name: string }[]>([])

  useEffect(() => {
    if (!open) return
    apiFetch<{ id: number; code: string; name: string }[]>('/legal-entities')
      .then((d) => setEntities(d ?? []))
      .catch(() => setEntities([]))
  }, [open])

  if (!open) return null
  const nameOk = values.name.trim().length > 0 && values.name.trim().length <= 64
  return (
    <Drawer title={values.id ? t.pages.department.editTitle : t.pages.department.createTitle} onClose={onClose}
      footer={
        <>
          <button className="org-btn" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="org-btn org-btn-primary" disabled={busy || !values.legalEntityId || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="org-form">
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.department.fLegalEntity}</label>
          <select className="org-select" value={values.legalEntityId ? String(values.legalEntityId) : ''}
            onChange={(e) => onChange({ ...values, legalEntityId: Number(e.target.value) || 0 })}>
            <option value="">{t.pages.department.pLegalEntity}</option>
            {entities.map((e) => <option key={e.id} value={e.id}>{e.code} {e.name}</option>)}
          </select>
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.department.fName}</label>
          <input className="org-input" value={values.name} placeholder={t.pages.department.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="text-[11px] text-[var(--color-danger)]">{t.pages.department.eName}</span>}
        </div>
        {submitError && <div className="org-error" style={{ margin: 0 }}>{submitError}</div>}
      </div>
    </Drawer>
  )
}
