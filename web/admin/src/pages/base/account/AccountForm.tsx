// 新建/编辑账号 Drawer 表单:三分组(基本信息/组织归属/数据范围),Pro 惯例。
// 级联数据源: /legal-entities、/departments?legalEntityId=、/posts?deptId=、/regions;
// 角色源: /menu-perms matrix.roleColumns(/roles 落地后切换)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { cascadeReset, validateAccount, type AccountFormValues } from './form'

const GROUP_TITLE_CLS = 'my-1.5 -mb-1 text-xs font-semibold tracking-wide text-[var(--shell-group-title)]'
const ERR_CLS = 'text-[11px] text-[var(--color-danger)]'

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
    <div className="flex flex-col gap-3.5">
      <div className={GROUP_TITLE_CLS}>{t.pages.account.gBasic}</div>
      <div className="flex flex-col gap-1.5">
        <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.account.fUsername}</label>
        <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.username} placeholder={t.pages.account.pUsername}
          onChange={(e) => set({ username: e.target.value })} />
        {err('invalidUsername') && <span className={ERR_CLS}>{t.pages.account.eUsername}</span>}
      </div>
      <div className="flex flex-col gap-1.5">
        <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.account.fPassword}</label>
        <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="password" value={values.password}
          placeholder={values.id ? t.pages.account.pPasswordEdit : t.pages.account.pPassword}
          onChange={(e) => set({ password: e.target.value })} />
        {err('shortPassword') && <span className={ERR_CLS}>{t.pages.account.ePassword}</span>}
      </div>
      <div className="flex flex-col gap-1.5">
        <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.account.fRealName}</label>
        <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.realName} placeholder={t.pages.account.pRealName}
          onChange={(e) => set({ realName: e.target.value })} />
        {err('invalidRealName') && <span className={ERR_CLS}>{t.pages.account.eRealName}</span>}
      </div>
      <div className="flex flex-col gap-1.5">
        <label>{t.pages.account.fPhone}</label>
        <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={values.phone} placeholder={t.pages.account.pPhone}
          onChange={(e) => set({ phone: e.target.value })} />
        {err('invalidPhone') && <span className={ERR_CLS}>{t.pages.account.ePhone}</span>}
      </div>
      <div className="flex flex-col gap-1.5">
        <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.pages.account.fRole}</label>
        <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={values.roleCode} onChange={(e) => set({ roleCode: e.target.value })}>
          <option value="">{t.pages.account.pRole}</option>
          {roles.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        {err('roleRequired') && <span className={ERR_CLS}>{t.pages.account.eRole}</span>}
      </div>

      <div className={GROUP_TITLE_CLS}>{t.pages.account.gOrg}</div>
      <div className="flex flex-col gap-1.5">
        <label>{t.pages.account.fLegalEntity}</label>
        <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={values.legalEntityId ? String(values.legalEntityId) : ''}
          onChange={(e) => onChange(cascadeReset({ ...values, legalEntityId: Number(e.target.value) || 0 }, 'legalEntityId'))}>
          <option value="">{t.pages.account.pAny}</option>
          {legalEntities.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <label>{t.pages.account.fDept}</label>
        <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={values.deptId ? String(values.deptId) : ''} disabled={!values.legalEntityId}
          onChange={(e) => onChange(cascadeReset({ ...values, deptId: Number(e.target.value) || 0 }, 'deptId'))}>
          <option value="">{t.pages.account.pAny}</option>
          {departments.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>
      <div className="flex flex-col gap-1.5">
        <label>{t.pages.account.fPost}</label>
        <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={values.postId ? String(values.postId) : ''} disabled={!values.deptId}
          onChange={(e) => set({ postId: Number(e.target.value) || 0 })}>
          <option value="">{t.pages.account.pAny}</option>
          {posts.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
      </div>

      <div className={GROUP_TITLE_CLS}>{t.pages.account.gScope}</div>
      <div className="flex flex-col gap-1.5">
        <label>{t.pages.account.fRegionScope}</label>
        <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={values.regionScope} onChange={(e) => set({ regionScope: e.target.value })}>
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
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <AccountForm values={values} onChange={onChange} errors={validateAccount(values, isEdit)} />
      {submitError && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]" style={{ marginTop: 12 }}>{submitError}</div>}
    </Drawer>
  )
}
