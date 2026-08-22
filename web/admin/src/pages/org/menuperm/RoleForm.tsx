// 新建/编辑角色 Drawer:名称 + 模板复用(内置角色整体复制)+ 权限勾选(可单独调整)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import {
  applyTemplate, rolePayload, splitPerms,
  type PermissionRow, type RoleDetail,
} from './roles'

export interface RoleFormState {
  id: number
  name: string
  codes: string[]
}

export function emptyRoleForm(): RoleFormState {
  return { id: 0, name: '', codes: [] }
}

export function roleToForm(r: RoleDetail): RoleFormState {
  return { id: r.id, name: r.name, codes: [...r.permissionCodes] }
}

function toggle(list: string[], code: string): string[] {
  return list.includes(code) ? list.filter((c) => c !== code) : [...list, code]
}

function PermGroup({ title, perms, codes, onToggle }: {
  title: string; perms: PermissionRow[]; codes: string[]; onToggle: (code: string) => void
}) {
  if (!perms.length) return null
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs font-medium text-[var(--shell-group-title)]">{title}</span>
      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 md:grid-cols-3">
        {perms.map((p) => (
          <label key={p.code} className="flex cursor-pointer items-center gap-1.5 text-[12px] text-[var(--shell-content-text)]">
            <input type="checkbox" className="cursor-pointer accent-[var(--shell-fab-bg)]" checked={codes.includes(p.code)} onChange={() => onToggle(p.code)} />
            <span className="truncate" title={`${p.name} (${p.code})`}>{p.name}</span>
          </label>
        ))}
      </div>
    </div>
  )
}

export function RoleFormDrawer({ open, state, builtinTemplates, onChange, onClose, onSubmit, busy, submitError }: {
  open: boolean
  state: RoleFormState
  builtinTemplates: RoleDetail[]
  onChange: (v: RoleFormState) => void
  onClose: () => void
  onSubmit: () => void
  busy: boolean
  submitError: string
}) {
  const t = useT()
  const [perms, setPerms] = useState<PermissionRow[]>([])

  useEffect(() => {
    if (!open) return
    apiFetch<PermissionRow[]>('/permissions')
      .then((d) => setPerms(d ?? []))
      .catch(() => setPerms([]))
  }, [open])

  if (!open) return null
  const tr = t.pages.menuperm
  const { menu, action } = splitPerms(perms)
  const nameOk = state.name.trim().length > 0 && state.name.trim().length <= 64
  const selected = new Set(state.codes)
  const onTemplate = (code: string) =>
    onChange({ ...state, codes: applyTemplate(builtinTemplates.find((r) => r.code === code) ?? null) })

  return (
    <Drawer title={state.id ? tr.editRole : tr.createRole} onClose={onClose}
      footer={
        <>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={onClose}>{t.pages.company.cancel}</button>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !nameOk} onClick={onSubmit}>
            {busy ? t.pages.account.submitting : t.pages.company.save}
          </button>
        </>
      }>
      <div className="flex flex-col gap-3.5">
        <div className="flex flex-col gap-1.5">
          <label><span className="mr-0.5 text-[var(--color-danger)]">*</span>{tr.roleName}</label>
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={state.name} placeholder={tr.roleNamePh} maxLength={64}
            onChange={(e) => onChange({ ...state, name: e.target.value })} />
        </div>
        {!state.id && (
          <div className="flex flex-col gap-1.5">
            <label>{tr.template}</label>
            <Dropdown
              value=""
              options={[{ value: '', label: tr.templateNone }, ...builtinTemplates.map((r) => ({ value: r.code, label: r.name }))]}
              onChange={onTemplate}
              ariaLabel={tr.template}
            />
            <span className="text-[11px] text-[var(--shell-crumb-text)]">{tr.templateHint}</span>
          </div>
        )}
        <PermGroup title={tr.permsMenu} perms={menu} codes={state.codes} onToggle={(c) => onChange({ ...state, codes: toggle(state.codes, c) })} />
        <PermGroup title={tr.permsAction} perms={action} codes={state.codes} onToggle={(c) => onChange({ ...state, codes: toggle(state.codes, c) })} />
        <span className="text-[11px] text-[var(--shell-crumb-text)]">{tr.permSelected.replace('{n}', String(selected.size))}</span>
        {submitError && <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{submitError}</div>}
      </div>
    </Drawer>
  )
}

export { rolePayload }
