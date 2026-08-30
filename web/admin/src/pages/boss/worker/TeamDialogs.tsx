// 装维队管理对话框集(000141):新建/编辑队伍、解散确认、成员调队;业绩统计走右侧抽屉。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { Button } from '../../../components/ui/button'
import { Input } from '../../../components/ui/input'
import type { TeamPerfRow, WorkerGroupRow, WorkerRow } from '../types'

export type DialogMode =
  | null
  | { type: 'create' }
  | { type: 'edit'; group: WorkerGroupRow }
  | { type: 'disband'; group: WorkerGroupRow }
  | { type: 'transfer'; worker: WorkerRow }
  | { type: 'perf'; group: WorkerGroupRow }

interface DialogsProps {
  mode: DialogMode
  groups: WorkerGroupRow[]
  workers: WorkerRow[]
  onClose: () => void
  onDone: () => void // 任一动作成功后刷新列表
}

// 表单控件统一 ui Input / ui Button;紧凑表单密度用 className 覆盖(h-8 px-4 text-[13px])。
const compact = 'h-8 px-4 text-[13px]'

// Modal 外壳:遮罩点击关闭 + 令牌化底色。
function Shell({ title, children, onClose }: { title: string; children: React.ReactNode; onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-page-modal flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="max-h-[85vh] w-[26rem] overflow-y-auto rounded-lg border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <h3 className="mb-4 text-base font-semibold text-[var(--shell-heading)]">{title}</h3>
        {children}
      </div>
    </div>
  )
}

function Err({ msg }: { msg: string }) {
  if (!msg) return null
  return <div className="mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{msg}</div>
}

// TeamForm 新建/编辑队伍;编辑时可指定队长(仅本队在职成员可选)。
function TeamForm({ group, workers, onClose, onDone }: { group: WorkerGroupRow | null; workers: WorkerRow[]; onClose: () => void; onDone: () => void }) {
  const w = useT().pages.workerPage
  const [name, setName] = useState(group?.name ?? '')
  const [code, setCode] = useState('')
  const [entityId, setEntityId] = useState('')
  const [leaderId, setLeaderId] = useState(group ? String(group.leaderId || '') : '')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const members = group ? workers.filter((x) => x.groupId === group.id && x.status === 1) : []

  const submit = async () => {
    if (busy) return
    if (!name.trim()) { setErr(w.eNameRequired); return }
    if (!group) {
      if (!code.trim()) { setErr(w.eCodeRequired); return }
      if (!/^\d+$/.test(entityId) || Number(entityId) <= 0) { setErr(w.eEntityRequired); return }
    }
    setBusy(true); setErr('')
    try {
      if (group) {
        await apiFetch(`/worker-groups/${group.id}`, { method: 'PUT', body: { name: name.trim(), leaderId: Number(leaderId) || 0 } })
      } else {
        await apiFetch('/worker-groups', { method: 'POST', body: { legalEntityId: Number(entityId), code: code.trim(), name: name.trim() } })
      }
      onClose(); onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  return (
    <div>
      <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.teamName}</label>
      <div className="mb-3"><Input value={name} onChange={(e) => setName(e.target.value)} /></div>
      {!group && (
        <>
          <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.teamCode}</label>
          <div className="mb-3"><Input value={code} onChange={(e) => setCode(e.target.value)} /></div>
          <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.teamEntity}</label>
          <div className="mb-3">
            <ResourcePicker
              value={entityId}
              onChange={setEntityId}
              load={() => apiFetch<{ id: number; name: string }[]>('/legal-entities').then((x) => x ?? [])}
              toOption={(x) => ({ value: String(x.id), label: x.name })}
              ariaLabel={w.teamEntity}
            />
          </div>
        </>
      )}
      {group && (
        <>
          <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.captain}</label>
          <div className="mb-3">
            <Dropdown
              value={leaderId}
              options={[{ value: '', label: w.captainEmpty }, ...members.map((m) => ({ value: String(m.id), label: `${m.name}(${m.staffNo})` }))]}
              onChange={setLeaderId}
              ariaLabel={w.captain}
            />
          </div>
        </>
      )}
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{w.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy} onClick={submit}>{w.save}</Button>
      </div>
    </div>
  )
}

// TransferForm 成员调队:目标队 + 原因。
function TransferForm({ worker, groups, onClose, onDone }: { worker: WorkerRow; groups: WorkerGroupRow[]; onClose: () => void; onDone: () => void }) {
  const w = useT().pages.workerPage
  const [groupId, setGroupId] = useState('')
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')

  const submit = async () => {
    if (busy || !groupId || Number(groupId) === worker.groupId) return
    setBusy(true); setErr('')
    try {
      await apiFetch(`/workers/${worker.id}/transfer`, { method: 'POST', body: { groupId: Number(groupId), reason: reason.trim() } })
      onClose(); onDone()
    } catch (e) {
      setErr(e instanceof Error ? e.message : w.actionFail)
    } finally { setBusy(false) }
  }

  return (
    <div>
      <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.transferTo}</label>
      <div className="mb-3">
        <Dropdown
          value={groupId}
          options={groups.filter((g) => g.id !== worker.groupId).map((g) => ({ value: String(g.id), label: g.name }))}
          onChange={setGroupId}
          ariaLabel={w.transferTo}
        />
      </div>
      <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.reasonLabel}</label>
      <div className="mb-3"><Input placeholder={w.reasonPlaceholder} value={reason} onChange={(e) => setReason(e.target.value)} /></div>
      <Err msg={err} />
      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className={compact} onClick={onClose}>{w.cancel}</Button>
        <Button size="sm" className={compact} disabled={busy || !groupId} onClick={submit}>{w.transfer}</Button>
      </div>
    </div>
  )
}

// PerfPanel 队伍月度业绩统计:period 切换即拉取。
function PerfPanel({ group }: { group: WorkerGroupRow }) {
  const w = useT().pages.workerPage
  const [period, setPeriod] = useState(() => new Date().toISOString().slice(0, 7))
  const [items, setItems] = useState<TeamPerfRow[]>([])
  const [err, setErr] = useState('')

  useEffect(() => {
    setErr('')
    apiFetch<{ period: string; items: TeamPerfRow[] }>(`/worker-groups/${group.id}/performance`, { query: { period } })
      .then((d) => setItems(d?.items ?? []))
      .catch((e) => setErr(e instanceof Error ? e.message : w.loadFail))
  }, [group.id, period]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div>
      <label className="mb-1 block text-[13px] text-[var(--shell-group-title)]">{w.periodLabel}</label>
      <div className="mb-3"><Input type="month" value={period} onChange={(e) => setPeriod(e.target.value)} /></div>
      <Err msg={err} />
      <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
        <thead><tr>{w.perfCols.map((x) => <th key={x} className="h-9 px-2 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
        <tbody>
          {items.map((m) => (
            <tr key={m.workerId}>
              <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{m.staffNo}</td>
              <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{m.name}</td>
              <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{m.finished}</td>
              <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{m.onTimeRate}%</td>
              <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{m.score}</td>
              <td className="h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)]">{m.isLeader ? w.captainTag : w.memberTag}</td>
            </tr>
          ))}
          {!items.length && !err && <tr><td colSpan={6} className="h-9 px-2 text-center text-[var(--shell-group-title)]">{w.empty}</td></tr>}
        </tbody>
      </table>
    </div>
  )
}

// TeamDialogs 对话框路由:按 mode 渲染对应表单/面板。
export function TeamDialogs({ mode, groups, workers, onClose, onDone }: DialogsProps) {
  const w = useT().pages.workerPage
  if (!mode) return null

  const disband = async (id: number) => {
    try {
      await apiFetch(`/worker-groups/${id}`, { method: 'DELETE' })
      onClose(); onDone()
    } catch (e) {
      alert(e instanceof Error ? e.message : w.actionFail)
    }
  }

  switch (mode.type) {
    case 'create':
    case 'edit':
      return <Shell title={mode.type === 'create' ? w.newTeam : w.editTeam} onClose={onClose}>
        <TeamForm group={mode.type === 'edit' ? mode.group : null} workers={workers} onClose={onClose} onDone={onDone} />
      </Shell>
    case 'transfer':
      return <Shell title={w.transferTitle} onClose={onClose}>
        <TransferForm worker={mode.worker} groups={groups} onClose={onClose} onDone={onDone} />
      </Shell>
    case 'perf':
      return <Drawer title={`${w.perfTitle} · ${mode.group.name}`} onClose={onClose}>
        <PerfPanel group={mode.group} />
      </Drawer>
    case 'disband':
      return <Shell title={w.disband} onClose={onClose}>
        <p className="mb-4 text-[13px] text-[var(--shell-content-text)]">{w.disbandConfirmText}</p>
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" className={compact} onClick={onClose}>{w.cancel}</Button>
          <Button variant="destructive" size="sm" className={compact} onClick={() => disband(mode.group.id)}>{w.confirmDisband}</Button>
        </div>
      </Shell>
  }
}
