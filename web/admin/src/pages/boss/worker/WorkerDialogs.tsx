// 师傅账号对话框集(2026-09-01 后台录入师傅):新增师傅(含师傅端登录密码)/重置密码。
// 表单密度与视觉复用 TeamDialogs 的 Shell/Err/compact;师傅端用手机号+密码登录。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Dropdown } from '../../../components/Dropdown'
import { MultiSelect } from '../../../components/MultiSelect'
import { Button } from '../../../components/ui/button'
import { Input } from '../../../components/ui/input'
import type { WorkerGroupRow } from '../types'
import { Err, Shell, compact } from './TeamDialogs'

export type WorkerDialogMode =
  | null
  | { type: 'create' }
  | { type: 'resetPwd'; workerId: number; name: string }

interface DialogsProps {
  mode: WorkerDialogMode
  groups: WorkerGroupRow[]
  onClose: () => void
  onDone: () => void
}

export interface RegionOption { id: number; name: string }

/** loadRegionOptions 经营区域下拉数据源(GET /regions 裸数组)。 */
export async function loadRegionOptions(): Promise<RegionOption[]> {
  const x = await apiFetch<RegionOption[]>('/regions')
  return Array.isArray(x) ? x : []
}

// WorkerForm 新增师傅:工号/姓名/手机号(登录名)/装维队/负责区域(多选,首个为主区域)/登录密码。
function WorkerForm({ groups, onClose, onDone }: { groups: WorkerGroupRow[]; onClose: () => void; onDone: () => void }) {
  const w = useT().pages.workerPage
  const [staffNo, setStaffNo] = useState('')
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [groupId, setGroupId] = useState('')
  const [regionIds, setRegionIds] = useState<string[]>([])
  const [regions, setRegions] = useState<RegionOption[]>([])
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  useEffect(() => { loadRegionOptions().then(setRegions).catch(() => setRegions([])) }, [])

  const submit = async () => {
    if (busy) return
    if (!staffNo.trim()) { setErr(w.eStaffNoRequired); return }
    if (!name.trim()) { setErr(w.eWorkerNameRequired); return }
    if (!phone.trim()) { setErr(w.ePhoneRequired); return }
    if (!groupId) { setErr(w.eGroupRequired); return }
    if (!regionIds.length) { setErr(w.eRegionsRequired); return }
    if (password.length < 6) { setErr(w.ePasswordShort); return }
    setBusy(true); setErr('')
    try {
      await apiFetch('/workers', {
        method: 'POST',
        body: {
          staffNo: staffNo.trim(), name: name.trim(), phone: phone.trim(),
          groupId: Number(groupId), regionIds: regionIds.map(Number), password,
        },
      })
      toast.success(w.toastWorkerSaved)
      onClose(); onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  const label = 'mb-1 block text-[13px] text-[var(--shell-group-title)]'
  return (
    <div>
      <label className={label}>{w.workerStaffNo}</label>
      <div className="mb-3"><Input value={staffNo} onChange={(e) => setStaffNo(e.target.value)} placeholder="WK-1024" /></div>
      <label className={label}>{w.workerName}</label>
      <div className="mb-3"><Input value={name} onChange={(e) => setName(e.target.value)} /></div>
      <label className={label}>{w.workerPhone}</label>
      <div className="mb-3"><Input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="13800000000" /></div>
      <label className={label}>{w.workerGroup}</label>
      <div className="mb-3">
        <Dropdown
          value={groupId}
          options={groups.map((g) => ({ value: String(g.id), label: g.name }))}
          onChange={setGroupId}
          ariaLabel={w.workerGroup}
        />
      </div>
      <label className={label}>{w.workerRegion}</label>
      <div className="mb-3">
        <MultiSelect
          values={regionIds}
          options={regions.map((r) => ({ value: String(r.id), label: r.name }))}
          onChange={setRegionIds}
          ariaLabel={w.workerRegion}
          placeholder={w.workerRegion}
          searchPlaceholder={w.searchPlaceholder}
        />
      </div>
      <p className="mb-3 text-xs text-[var(--shell-group-title)]">{w.workerRegionHint}</p>
      <label className={label}>{w.workerPassword}</label>
      <div className="mb-1"><Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
      <p className="mb-3 text-xs text-[var(--shell-group-title)]">{w.workerPasswordHint}</p>
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{w.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy} onClick={submit}>{w.save}</Button>
      </div>
    </div>
  )
}

// ResetPwdForm 重置师傅登录密码:重置后师傅端需用新密码登录。
function ResetPwdForm({ workerId, name, onClose }: { workerId: number; name: string; onClose: () => void }) {
  const w = useT().pages.workerPage
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const submit = async () => {
    if (busy) return
    if (password.length < 6) { setErr(w.ePasswordShort); return }
    setBusy(true); setErr('')
    try {
      await apiFetch(`/workers/${workerId}/password`, { method: 'PUT', body: { password } })
      toast.success(w.toastPwdReset)
      onClose()
    } catch (e) {
      setErr(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  return (
    <div>
      <p className="mb-3 text-[13px] text-[var(--shell-content-text)]">
        {w.resetPwdConfirmText.replace('{name}', name)}
      </p>
      <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.workerPassword}</label>
      <div className="mb-3"><Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></div>
      <p className="mb-3 text-xs text-[var(--shell-group-title)]">{w.workerPasswordHint}</p>
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{w.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy} onClick={submit}>{w.save}</Button>
      </div>
    </div>
  )
}

// WorkerDialogs 对话框路由:按 mode 渲染新增师傅/重置密码。
export function WorkerDialogs({ mode, groups, onClose, onDone }: DialogsProps) {
  const w = useT().pages.workerPage
  if (!mode) return null
  if (mode.type === 'create') {
    return <Shell title={w.newWorkerTitle} onClose={onClose}>
      <WorkerForm groups={groups} onClose={onClose} onDone={onDone} />
    </Shell>
  }
  return <Shell title={w.resetPwdTitle} onClose={onClose}>
    <ResetPwdForm workerId={mode.workerId} name={mode.name} onClose={onClose} />
  </Shell>
}
