// 新建/编辑账号 Drawer 表单:三分组(基本信息/组织归属/数据范围),Pro 惯例。
// 级联数据源: /legal-entities、/departments?legalEntityId=、/posts?deptId=、/regions;
// 角色源: /menu-perms matrix.roleColumns(/roles 落地后切换)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { Drawer } from '../../../components/Drawer'
import { Input } from '../../../components/ui/input'
import { FormField } from '../../../components/business/form-field'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SubmitButton, type SubmitState } from '../../../components/business/submit-button'
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
      <FormField label={t.pages.account.fUsername} required>
        <Input value={values.username} placeholder={t.pages.account.pUsername}
          onChange={(e) => set({ username: e.target.value })} />
        {err('invalidUsername') && <span className={ERR_CLS}>{t.pages.account.eUsername}</span>}
      </FormField>
      <FormField label={t.pages.account.fPassword} required>
        <Input type="password" value={values.password}
          placeholder={values.id ? t.pages.account.pPasswordEdit : t.pages.account.pPassword}
          onChange={(e) => set({ password: e.target.value })} />
        {err('shortPassword') && <span className={ERR_CLS}>{t.pages.account.ePassword}</span>}
      </FormField>
      <FormField label={t.pages.account.fRealName} required>
        <Input value={values.realName} placeholder={t.pages.account.pRealName}
          onChange={(e) => set({ realName: e.target.value })} />
        {err('invalidRealName') && <span className={ERR_CLS}>{t.pages.account.eRealName}</span>}
      </FormField>
      <FormField label={t.pages.account.fPhone}>
        <Input value={values.phone} placeholder={t.pages.account.pPhone}
          onChange={(e) => set({ phone: e.target.value })} />
        {err('invalidPhone') && <span className={ERR_CLS}>{t.pages.account.ePhone}</span>}
      </FormField>
      <FormField label={t.pages.account.fRole} required>
        <Dropdown
          value={values.roleCode}
          options={[{ value: '', label: t.pages.account.pRole }, ...roles]}
          onChange={(v) => set({ roleCode: v })}
          ariaLabel={t.pages.account.fRole}
        />
        {err('roleRequired') && <span className={ERR_CLS}>{t.pages.account.eRole}</span>}
      </FormField>

      <div className={GROUP_TITLE_CLS}>{t.pages.account.gOrg}</div>
      <FormField label={t.pages.account.fLegalEntity}>
        <Dropdown
          value={values.legalEntityId ? String(values.legalEntityId) : ''}
          options={[{ value: '', label: t.pages.account.pAny }, ...legalEntities]}
          onChange={(v) => onChange(cascadeReset({ ...values, legalEntityId: Number(v) || 0 }, 'legalEntityId'))}
          ariaLabel={t.pages.account.fLegalEntity}
        />
      </FormField>
      <FormField label={t.pages.account.fDept}>
        <Dropdown
          value={values.deptId ? String(values.deptId) : ''}
          options={[{ value: '', label: t.pages.account.pAny }, ...departments]}
          onChange={(v) => onChange(cascadeReset({ ...values, deptId: Number(v) || 0 }, 'deptId'))}
          ariaLabel={t.pages.account.fDept}
          disabled={!values.legalEntityId}
        />
      </FormField>
      <FormField label={t.pages.account.fPost}>
        <Dropdown
          value={values.postId ? String(values.postId) : ''}
          options={[{ value: '', label: t.pages.account.pAny }, ...posts]}
          onChange={(v) => set({ postId: Number(v) || 0 })}
          ariaLabel={t.pages.account.fPost}
          disabled={!values.deptId}
        />
      </FormField>

      <div className={GROUP_TITLE_CLS}>{t.pages.account.gScope}</div>
      <FormField label={t.pages.account.fRegionScope}>
        <Dropdown
          value={values.regionScope}
          options={[{ value: '', label: t.pages.account.scopeAll }, ...regions]}
          onChange={(v) => set({ regionScope: v })}
          ariaLabel={t.pages.account.fRegionScope}
        />
      </FormField>
    </div>
  )
}

export function AccountFormDrawer({
  open, values, onChange, onClose, onSubmit, busy, submitError, submitState,
}: {
  open: boolean
  values: AccountFormValues
  onChange: (v: AccountFormValues) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
  submitState: SubmitState
}) {
  const t = useT()
  const isEdit = Boolean(values.id)
  if (!open) return null
  return (
    <Drawer title={isEdit ? t.pages.account.editTitle : t.pages.account.createTitle} onClose={onClose}
      footer={
        <>
          <ToolbarButton onClick={onClose} disabled={busy}>{t.pages.company.cancel}</ToolbarButton>
          <SubmitButton
            state={submitState}
            labels={{
              idle: t.pages.company.save,
              loading: t.common.loading,
              success: t.common.saveOk,
              failed: t.common.saveFail,
            }}
            disabled={busy}
            onClick={onSubmit}
          />
        </>
      }>
      <AccountForm values={values} onChange={onChange} errors={validateAccount(values, isEdit)} />
      {submitError && <div className="px-4 pb-4"><ErrorBanner message={submitError} /></div>}
    </Drawer>
  )
}
