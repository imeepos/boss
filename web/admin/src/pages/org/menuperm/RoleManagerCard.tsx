// 角色管理卡片:内置(只读)+ 派生角色(新建/编辑/删除);模板=内置角色权限集。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useConfirm } from '../../../components/ConfirmDialog'
import { EmptyState } from '../../../components/business'
import { roleErrorCode, toRoleRows, type RoleDetail } from './roles'
import { emptyRoleForm, rolePayload, roleToForm, RoleFormDrawer, type RoleFormState } from './RoleForm'

export function RoleManagerCard({ onChanged }: { onChanged: () => void }) {
  const t = useT()
  const tr = t.pages.menuperm
  const confirmDialog = useConfirm()
  const [roles, setRoles] = useState<RoleDetail[]>([])
  const [error, setError] = useState('')
  const [form, setForm] = useState<RoleFormState | null>(null)
  const [formError, setFormError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    apiFetch<RoleDetail[]>('/role-details')
      .then((d) => setRoles(toRoleRows(d ?? [])))
      .catch((e) => setError(e instanceof Error ? e.message : tr.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const builtin = roles.filter((r) => r.isBuiltin)
  const errText = (e: unknown, fallback: string): string => {
    const kind = roleErrorCode(e)
    if (kind === 'protected') return tr.errProtected
    if (kind === 'inUse') return tr.errInUse
    return e instanceof Error ? e.message : fallback
  }

  const submit = async () => {
    if (!form || busy) return
    setBusy(true)
    setFormError('')
    try {
      const body = rolePayload(form.name, form.codes)
      if (form.id) await apiFetch(`/roles/${form.id}`, { method: 'PUT', body })
      else await apiFetch('/roles', { method: 'POST', body })
      setForm(null)
      load()
      onChanged()
    } catch (e) {
      setFormError(errText(e, tr.saveFail))
    } finally {
      setBusy(false)
    }
  }

  const remove = async (r: RoleDetail) => {
    if (!(await confirmDialog(tr.deleteConfirm.replace('{name}', r.name), { danger: true }))) return
    try {
      await apiFetch(`/roles/${r.id}`, { method: 'DELETE' })
      load()
      onChanged()
    } catch (e) {
      setError(errText(e, tr.deleteFail))
    }
  }

  return (
    <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <div className="text-[15px] font-semibold text-[var(--shell-heading)]">{tr.roleTitle}</div>
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setFormError(''); setForm(emptyRoleForm()) }}>{tr.createRole}</button>
      </div>
      {error ? (
        <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
      ) : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead>
              <tr>
                {[tr.roleName, tr.roleCode, tr.roleType, tr.permCount, tr.actionColumn].map((c) => (
                  <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {roles.map((r) => (
                <tr key={r.id}>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.name}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.code}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <span className={r.isBuiltin ? 'text-[var(--shell-group-title)]' : 'font-medium text-[var(--shell-heading)]'}>{r.isBuiltin ? tr.builtin : tr.custom}</span>
                  </td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.permissionCodes.length}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                    {r.isBuiltin ? (
                      <span className="text-[var(--shell-crumb-text)]" title={tr.builtinReadOnly}>—</span>
                    ) : (
                      <span className="inline-flex items-center">
                        <button onClick={() => { setFormError(''); setForm(roleToForm(r)) }}>{t.pages.account.edit}</button>
                        <span className="text-[var(--shell-side-border)]">|</span>
                        <button className="text-[var(--color-danger)]" onClick={() => remove(r)}>{tr.deleteRole}</button>
                      </span>
                    )}
                  </td>
                </tr>
              ))}
              {!roles.length && <tr><td colSpan={5} className="px-3 py-6"><EmptyState text={tr.empty} /></td></tr>}
            </tbody>
          </table>
        </div>
      )}
      <RoleFormDrawer
        open={form !== null}
        state={form ?? emptyRoleForm()}
        builtinTemplates={builtin}
        onChange={setForm}
        onClose={() => setForm(null)}
        onSubmit={submit}
        busy={busy}
        submitError={formError}
      />
    </div>
  )
}
