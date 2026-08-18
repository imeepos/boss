// 岗位新建/编辑 Drawer 表单:归属部门 + 代码/名称 + 绑定角色(多选,全量替换)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
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
          <button className="org-btn" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="org-btn org-btn-primary" disabled={busy || !values.deptId || !codeOk || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="org-form">
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.post.fDept}</label>
          <select className="org-select" value={values.deptId ? String(values.deptId) : ''}
            onChange={(e) => onChange({ ...values, deptId: Number(e.target.value) || 0 })}>
            <option value="">{t.pages.post.pDept}</option>
            {depts.map((d) => (
              <option key={d.id} value={d.id}>{d.legalEntity ? `${d.legalEntity} / ` : ''}{d.name}</option>
            ))}
          </select>
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.post.fCode}</label>
          <input className="org-input" value={values.code} placeholder={t.pages.post.pCode}
            onChange={(e) => onChange({ ...values, code: e.target.value })} />
          {!codeOk && values.code !== '' && <span className="acc-err">{t.pages.post.eCode}</span>}
        </div>
        <div className="org-field">
          <label><span className="req">*</span>{t.pages.post.fName}</label>
          <input className="org-input" value={values.name} placeholder={t.pages.post.pName}
            onChange={(e) => onChange({ ...values, name: e.target.value })} />
          {!nameOk && values.name !== '' && <span className="acc-err">{t.pages.post.eName}</span>}
        </div>
        <div className="org-field">
          <label>{t.pages.post.fRoles}</label>
          <div className="post-roles">
            {roles.map((r) => (
              <label key={r.code} className="post-role-chip">
                <input type="checkbox" checked={values.roles.includes(r.code)} onChange={() => toggleRole(r.code)} />
                {r.name}
              </label>
            ))}
          </div>
        </div>
        {submitError && <div className="org-error" style={{ margin: 0 }}>{submitError}</div>}
      </div>
    </Drawer>
  )
}
