// 新建/编辑账号 Drawer 表单:三分组(基本信息/组织归属/数据范围),Pro 惯例。
// 级联数据源: /legal-entities、/departments?legalEntityId=、/posts?deptId=、/regions;
// 角色源: /menu-perms matrix.roleColumns(/roles 落地后切换)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { cascadeReset, validateAccount, type AccountFormValues } from './form'
import '../../org/org.css'
import './account.css'

interface Option { value: string; label: string }

export interface AccountFormHandlers {
  values: AccountFormValues
  onChange: (v: AccountFormValues) => void
  errors: string[]
}

export function AccountForm({ values, onChange, errors }: AccountFormHandlers) {
  const t = useT()
  const [legalEntities, setLegalEntities] = useState<Option[]>([])
  const [departments, setDepartments] = useState<Option[]>([])
  const [posts, setPosts] = useState<Option[]>([])
  const [regions, setRegions] = useState<Option[]>([])
  const [roles, setRoles] = useState<Option[]>([])

  useEffect(() => {
    apiFetch<{ id: number; code: string; name: string }[]>('/legal-entities')
      .then((d) => setLegalEntities((d ?? []).map((e) => ({ value: String(e.id), label: `${e.code} ${e.name}` }))))
      .catch(() => setLegalEntities([]))
    apiFetch<{ id: number; path: string; name: string }[]>('/regions')
      .then((d) => setRegions((d ?? []).map((r) => ({ value: r.path, label: `${r.name} (${r.path})` }))))
      .catch(() => setRegions([]))
    apiFetch<{ code: string; name: string }[]>('/roles')
      .then((d) => {
        if (d?.length) {
          setRoles(d.map((c) => ({ value: c.code, label: c.name })))
          return
        }
        // /roles 空兜底:借 menu-perms 角色列(同源 roles 表)。
        return apiFetch<{ matrix: { roleColumns: { roleCode: string; roleName: string }[] } }>('/menu-perms')
          .then((m) => setRoles((m?.matrix?.roleColumns ?? []).map((c) => ({ value: c.roleCode, label: c.roleName }))))
      })
      .catch(() => setRoles([]))
  }, [])

  useEffect(() => {
    if (!values.legalEntityId) { setDepartments([]); return }
    apiFetch<{ id: number; name: string }[]>('/departments', { query: { legalEntityId: values.legalEntityId } })
      .then((d) => setDepartments((d ?? []).map((x) => ({ value: String(x.id), label: x.name }))))
      .catch(() => setDepartments([]))
  }, [values.legalEntityId])

  useEffect(() => {
    if (!values.deptId) { setPosts([]); return }
    apiFetch<{ id: number; code: string; name: string }[]>('/posts', { query: { deptId: values.deptId } })
      .then((d) => setPosts((d ?? []).map((x) => ({ value: String(x.id), label: `${x.code} ${x.name}` }))))
      .catch(() => setPosts([]))
  }, [values.deptId])

  const err = (k: string) => errors.includes(k)
  const set = (patch: Partial<AccountFormValues>) => onChange({ ...values, ...patch })

  return (
    <div className="org-form">
      <div className="acc-group-title">{t.pages.account.gBasic}</div>
      <div className="org-field">
        <label><span className="req">*</span>{t.pages.account.fUsername}</label>
        <input className="org-input" value={values.username} placeholder={t.pages.account.pUsername}
          onChange={(e) => set({ username: e.target.value })} />
        {err('invalidUsername') && <span className="acc-err">{t.pages.account.eUsername}</span>}
      </div>
      <div className="org-field">
        <label><span className="req">*</span>{t.pages.account.fPassword}</label>
        <input className="org-input" type="password" value={values.password}
          placeholder={values.id ? t.pages.account.pPasswordEdit : t.pages.account.pPassword}
          onChange={(e) => set({ password: e.target.value })} />
        {err('shortPassword') && <span className="acc-err">{t.pages.account.ePassword}</span>}
      </div>
      <div className="org-field">
        <label><span className="req">*</span>{t.pages.account.fRealName}</label>
        <input className="org-input" value={values.realName} placeholder={t.pages.account.pRealName}
          onChange={(e) => set({ realName: e.target.value })} />
        {err('invalidRealName') && <span className="acc-err">{t.pages.account.eRealName}</span>}
      </div>
      <div className="org-field">
        <label>{t.pages.account.fPhone}</label>
        <input className="org-input" value={values.phone} placeholder={t.pages.account.pPhone}
          onChange={(e) => set({ phone: e.target.value })} />
        {err('invalidPhone') && <span className="acc-err">{t.pages.account.ePhone}</span>}
      </div>
      <div className="org-field">
        <label><span className="req">*</span>{t.pages.account.fRole}</label>
        <select className="org-select" value={values.roleCode} onChange={(e) => set({ roleCode: e.target.value })}>
          <option value="">{t.pages.account.pRole}</option>
          {roles.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        {err('roleRequired') && <span className="acc-err">{t.pages.account.eRole}</span>}
      </div>

      <div className="acc-group-title">{t.pages.account.gOrg}</div>
      <div className="org-field">
        <label>{t.pages.account.fLegalEntity}</label>
        <select className="org-select" value={values.legalEntityId ? String(values.legalEntityId) : ''}
          onChange={(e) => onChange(cascadeReset({ ...values, legalEntityId: Number(e.target.value) || 0 }, 'legalEntityId'))}>
          <option value="">{t.pages.account.pAny}</option>
          {legalEntities.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>
      <div className="org-field">
        <label>{t.pages.account.fDept}</label>
        <select className="org-select" value={values.deptId ? String(values.deptId) : ''} disabled={!values.legalEntityId}
          onChange={(e) => onChange(cascadeReset({ ...values, deptId: Number(e.target.value) || 0 }, 'deptId'))}>
          <option value="">{t.pages.account.pAny}</option>
          {departments.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>
      <div className="org-field">
        <label>{t.pages.account.fPost}</label>
        <select className="org-select" value={values.postId ? String(values.postId) : ''} disabled={!values.deptId}
          onChange={(e) => set({ postId: Number(e.target.value) || 0 })}>
          <option value="">{t.pages.account.pAny}</option>
          {posts.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>

      <div className="acc-group-title">{t.pages.account.gScope}</div>
      <div className="org-field">
        <label>{t.pages.account.fRegionScope}</label>
        <select className="org-select" value={values.regionScope} onChange={(e) => set({ regionScope: e.target.value })}>
          <option value="">{t.pages.account.scopeAll}</option>
          {regions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>
    </div>
  )
}

export function AccountFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError,
}: {
  open: boolean
  values: AccountFormValues
  onChange: (v: AccountFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const isEdit = Boolean(values.id)
  if (!open) return null
  return (
    <Drawer title={isEdit ? t.pages.account.editTitle : t.pages.account.createTitle} onClose={onClose}
      footer={
        <>
          <button className="org-btn" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="org-btn org-btn-primary" disabled={busy} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <AccountForm values={values} onChange={onChange} errors={validateAccount(values, isEdit)} />
      {submitError && <div className="org-error" style={{ marginTop: 12 }}>{submitError}</div>}
    </Drawer>
  )
}
