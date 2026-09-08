// 角色管理卡片:内置(只读)+ 派生角色(新建/编辑/删除);模板=内置角色权限集。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { useConfirm } from '../../../components/ConfirmDialog'
import { EmptyState } from '../../../components/business'
import { ErrorBanner } from '../../../components/business/page-head'
import { Card } from '../../../components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
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
      toast.success(tr.roleSaveOk)
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
      toast.success(tr.roleDelOk)
      load()
      onChanged()
    } catch (e) {
      toast.error(errText(e, tr.deleteFail))
    }
  }

  const act = 'cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--shell-content-text)] underline-offset-2 hover:text-[var(--shell-heading)] hover:underline disabled:cursor-not-allowed disabled:opacity-60'
  const actDanger = 'cursor-pointer border-none bg-none p-0 text-[13px] text-[var(--color-danger)] underline-offset-2 hover:underline disabled:cursor-not-allowed disabled:opacity-60'

  return (
    <Card>
      <div className="flex flex-wrap items-center gap-2 p-4">
        <div className="text-[15px] font-semibold text-[var(--shell-heading)]">{tr.roleTitle}</div>
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
        <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setFormError(''); setForm(emptyRoleForm()) }}>{tr.createRole}</button>
      </div>
      {error ? (
        <div className="mx-4 mb-3"><ErrorBanner message={error} /></div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              {[tr.roleName, tr.roleCode, tr.roleType, tr.permCount, tr.actionColumn].map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {roles.map((r) => (
              <TableRow key={r.id}>
                <TableCell>{r.name}</TableCell>
                <TableCell>{r.code}</TableCell>
                <TableCell>
                  <span className={r.isBuiltin ? 'text-[var(--shell-group-title)]' : 'font-medium text-[var(--shell-heading)]'}>{r.isBuiltin ? tr.builtin : tr.custom}</span>
                </TableCell>
                <TableCell>{r.permissionCodes.length}</TableCell>
                <TableCell>
                  {r.isBuiltin ? (
                    <span className="text-[var(--shell-crumb-text)]" title={tr.builtinReadOnly}>—</span>
                  ) : (
                    <span className="inline-flex items-center gap-1.5">
                      <button className={act} disabled={busy} onClick={() => { setFormError(''); setForm(roleToForm(r)) }}>{t.pages.account.edit}</button>
                      <span className="text-[var(--shell-side-border)]">|</span>
                      <button className={actDanger} disabled={busy} onClick={() => remove(r)}>{tr.deleteRole}</button>
                    </span>
                  )}
                </TableCell>
              </TableRow>
            ))}
            {!roles.length && (
              <TableRow><TableCell colSpan={5} className="px-3 py-6"><EmptyState text={tr.empty} /></TableCell></TableRow>
            )}
          </TableBody>
        </Table>
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
    </Card>
  )
}
