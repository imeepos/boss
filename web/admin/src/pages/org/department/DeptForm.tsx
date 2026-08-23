// 部门新建/编辑 Drawer 表单:挂靠子公司 + 名称;子公司内同名唯一(40900)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner } from '../../../components/business/page-head'
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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !values.legalEntityId || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <FormField label={t.pages.department.fLegalEntity} required>
          <Dropdown
            value={values.legalEntityId ? String(values.legalEntityId) : ''}
            options={[
              { value: '', label: t.pages.department.pLegalEntity },
              ...entities.map((e) => ({ value: String(e.id), label: `${e.code} ${e.name}` })),
            ]}
            onChange={(v) => onChange({ ...values, legalEntityId: Number(v) || 0 })}
            ariaLabel={t.pages.department.pLegalEntity}
          />
        </FormField>
        <FormField label={t.pages.department.fName} required
          error={!nameOk && values.name !== '' ? t.pages.department.eName : undefined}>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.name} placeholder={t.pages.department.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
        </FormField>
        {submitError && <ErrorBanner message={submitError} />}
      </div>
    </Drawer>
  )
}
