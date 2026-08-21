// 岗位新建/编辑 Drawer 表单:归属部门 + 代码/名称 + 绑定角色(多选,全量替换)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import type { PostRow } from './filter'

export interface PostFormValues {
  id: number
  deptId: number
  code: string
  name: string
  roles: string[]
}

export function emptyPostForm(): PostFormValues {
  return { id: 0, deptId: 0, code: '', name: '', roles: [] }
}

export function rowToPostForm(r: PostRow): PostFormValues {
  return { id: r.id, deptId: r.deptId, code: r.code, name: r.name, roles: r.roles ?? [] }
}

export function PostFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: PostFormValues
  onChange: (v: PostFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const [depts, setDepts] = useState<{ id: number; legalEntity: string; name: string }[]>([])
  const [roles, setRoles] = useState<{ code: string; name: string }[]>([])

  useEffect(() => {
    if (!open) return
    apiFetch<{ id: number; legalEntity: string; name: string }[]>('/departments')
      .then((d) => setDepts(d ?? []))
      .catch(() => setDepts([]))
    apiFetch<{ code: string; name: string }[]>('/roles')
      .then((d) => setRoles(d ?? []))
      .catch(() => setRoles([]))
  }, [open])

  if (!open) return null
  const codeOk = /^[a-z][a-z0-9_]{1,63}$/.test(values.code)
  const nameOk = values.name.trim().length > 0 && values.name.trim().length <= 64
  const toggleRole = (code: string) => onChange({
    ...values,
    roles: values.roles.includes(code)
      ? values.roles.filter((r) => r !== code)
      : [...values.roles, code],
  })

  return (
    <Drawer title={values.id ? t.pages.post.editTitle : t.pages.post.createTitle} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !values.deptId || !codeOk || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.post.fDept}</label>
          <Dropdown
            value={values.deptId ? String(values.deptId) : ''}
            options={[
              { value: '', label: t.pages.post.pDept },
              ...depts.map((d) => ({ value: String(d.id), label: d.legalEntity ? `${d.legalEntity} / ${d.name}` : d.name })),
            ]}
            onChange={(v) => onChange({ ...values, deptId: Number(v) || 0 })}
            ariaLabel={t.pages.post.pDept}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.post.fCode}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.code} placeholder={t.pages.post.pCode}
            onChange={(e) => onChange({ ...values, code: e.target.value })} />
          {!codeOk && values.code !== '' && <span className="text-[11px] text-[var(--color-danger)]">{t.pages.post.eCode}</span>}
        </div>
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.post.fName}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.name} placeholder={t.pages.post.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="text-[11px] text-[var(--color-danger)]">{t.pages.post.eName}</span>}
        </div>
        <div className="flex flex-col gap-1.5">
          <label>{t.pages.post.fRoles}</label>
          <div className="flex flex-wrap gap-2">
            {roles.map((r) => (
              <label key={r.code} className="inline-flex cursor-pointer items-center gap-1.5 rounded-full border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-content-text)] [&:has(input:checked)]:border-[var(--shell-fab-bg)] [&:has(input:checked)]:bg-[var(--shell-fab-bg)] [&:has(input:checked)]:text-[var(--shell-fab-icon)]">
                <input type="checkbox" checked={values.roles.includes(r.code)} onChange={() => toggleRole(r.code)} />
                {r.name}
              </label>
            ))}
          </div>
        </div>
        {submitError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ margin: 0 }}>{submitError}</div>}
      </div>
    </Drawer>
  )
}
