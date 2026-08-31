// 企业员工后台录入对话框集(000172):录入员工(工号/登录名/密码/角色)/重置密码。
// 表单密度与交互复用师傅录入(WorkerDialogs)的模式;角色用 Dropdown 不用原生 select。
import { useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { Button } from '../../../components/ui/button'
import { Input } from '../../../components/ui/input'
import { Err, Shell, compact } from '../../boss/worker/TeamDialogs'

export type EntityStaffDialogMode =
  | null
  | { type: 'create'; entityId: number }
  | { type: 'resetPwd'; entityId: number; accountId: number; name: string }

// StaffForm 录入企业员工:工号选填(全局唯一),角色限企业管理员/企业员工。
function StaffForm({ entityId, onClose, onDone }: { entityId: number; onClose: () => void; onDone: () => void }) {
  const t = useT().pages.company.staff
  const [staffNo, setStaffNo] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [realName, setRealName] = useState('')
  const [phone, setPhone] = useState('')
  const [roleCode, setRoleCode] = useState('partner_staff')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const submit = async () => {
    if (busy) return
    if (!username.trim()) { setErr(t.eUsernameRequired); return }
    if (password.length < 6) { setErr(t.ePasswordShort); return }
    if (!realName.trim()) { setErr(t.eRealNameRequired); return }
    setBusy(true); setErr('')
    try {
      await apiFetch(`/legal-entities/${entityId}/staff`, {
        method: 'POST',
        body: {
          staffNo: staffNo.trim(), username: username.trim(), password,
          realName: realName.trim(), phone: phone.trim(), roleCode,
        },
      })
      onClose(); onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : t.actionFail)
    } finally { setBusy(false) }
  }

  const label = 'mb-1 block text-[13px] text-[var(--shell-group-title)]'
  return (
    <div>
      <label className={label}>{t.staffNo}</label>
      <div className="mb-3"><Input value={staffNo} onChange={(e) => setStaffNo(e.target.value)} placeholder={t.staffNoPh} /></div>
      <label className={label}><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.username}</label>
      <div className="mb-3"><Input value={username} onChange={(e) => setUsername(e.target.value)} placeholder={t.usernamePh} /></div>
      <label className={label}><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.password}</label>
      <div className="mb-1"><Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
      <p className="mb-3 text-xs text-[var(--shell-group-title)]">{t.passwordHint}</p>
      <label className={label}><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.realName}</label>
      <div className="mb-3"><Input value={realName} onChange={(e) => setRealName(e.target.value)} /></div>
      <label className={label}>{t.phone}</label>
      <div className="mb-3"><Input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="13800000000" /></div>
      <label className={label}><span className="mr-0.5 text-[var(--color-danger)]">*</span>{t.role}</label>
      <div className="mb-4">
        <Dropdown
          value={roleCode}
          options={[
            { value: 'partner_staff', label: t.roleStaff },
            { value: 'partner_admin', label: t.roleAdmin },
          ]}
          onChange={setRoleCode}
          ariaLabel={t.role}
        />
      </div>
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{t.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy} onClick={submit}>{t.save}</Button>
      </div>
    </div>
  )
}

// ResetPwdForm 重置企业员工登录密码:员工下次登录用新密码。
function ResetPwdForm({ entityId, accountId, name, onClose }: { entityId: number; accountId: number; name: string; onClose: () => void }) {
  const t = useT().pages.company.staff
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const submit = async () => {
    if (busy) return
    if (password.length < 6) { setErr(t.ePasswordShort); return }
    setBusy(true); setErr('')
    try {
      await apiFetch(`/legal-entities/${entityId}/staff/${accountId}/password`, { method: 'PUT', body: { password } })
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : t.actionFail)
    } finally { setBusy(false) }
  }

  return (
    <div>
      <p className="mb-3 text-[13px] text-[var(--shell-content-text)]">{t.resetPwdConfirmText.replace('{name}', name)}</p>
      <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{t.password}</label>
      <div className="mb-1"><Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
      <p className="mb-3 text-xs text-[var(--shell-group-title)]">{t.passwordHint}</p>
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{t.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy} onClick={submit}>{t.save}</Button>
      </div>
    </div>
  )
}

// EntityStaffDialogs 对话框路由:按 mode 渲染录入员工/重置密码。
export function EntityStaffDialogs({ mode, onClose, onDone }: { mode: EntityStaffDialogMode; onClose: () => void; onDone: () => void }) {
  const t = useT().pages.company.staff
  if (!mode) return null
  if (mode.type === 'create') {
    return <Shell title={t.createTitle} onClose={onClose}>
      <StaffForm entityId={mode.entityId} onClose={onClose} onDone={onDone} />
    </Shell>
  }
  return <Shell title={t.resetPwdTitle} onClose={onClose}>
    <ResetPwdForm entityId={mode.entityId} accountId={mode.accountId} name={mode.name} onClose={onClose} />
  </Shell>
}
