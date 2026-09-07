// 预算与里程碑面板(P-INFRA-1 W6/G2,000218;挂施工单详情)。
// 文案为字面量:沿用 ODN 施工页签先例,中央登记仅登记菜单项。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ToolbarButton } from '../../../components/business/page-head'
import { fmtMoney } from './constructions'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

interface Milestone {
  id: number
  projectId: number
  name: string
  plannedDate?: string
  status: string
  doneAt?: string
  createdAt: string
}

const M_VARIANT: Record<string, 'default' | 'success'> = { PENDING: 'default', DONE: 'success' }
const M_TEXT: Record<string, string> = { PENDING: '未完成', DONE: '已完成' }

export function BudgetMilestonePanel({ project, onChanged }: {
  project: { id: number; status: string; budgetAmount?: number | null; settledAmount?: number } | null
  onChanged: () => void
}) {
  const [rows, setRows] = useState<Milestone[]>([])
  const [budget, setBudget] = useState('')
  const [name, setName] = useState('')
  const [planned, setPlanned] = useState('')
  const [edits, setEdits] = useState<Record<number, { name: string; plannedDate: string }>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const pid = project?.id
  const status = project?.status
  const load = useCallback(async () => {
    if (!pid) return
    try { setRows((await apiFetch<Milestone[]>('/odn/constructions/' + pid + '/milestones')) ?? []) }
    catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [pid])
  useEffect(() => { void load() }, [load])
  useEffect(() => { setBudget(project?.budgetAmount != null ? String(project.budgetAmount) : '') }, [project?.budgetAmount])

  const act = async (fn: () => Promise<unknown>, okMsg: string) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await load(); onChanged() }
    catch (e) { setError(e instanceof Error ? e.message : '操作失败') } finally { setBusy(false) }
  }

  const saveBudget = () => {
    const raw = budget.trim()
    const v = raw === '' ? null : Number(raw)
    if (raw !== '' && (!Number.isFinite(v) || (v as number) < 0)) { setError('预算金额须为非负数或留空清除'); return }
    void act(() => apiFetch('/odn/constructions/' + pid + '/budget', { method: 'PUT', body: { budgetAmount: v } }), '预算已保存')
  }
  const add = () => {
    if (!name.trim()) { setError('里程碑名称必填'); return }
    void act(() => apiFetch('/odn/constructions/' + pid + '/milestones', { method: 'POST', body: {
      name: name.trim(), plannedDate: planned.trim() } }), '里程碑已追加').then(() => { setName(''); setPlanned('') })
  }
  const saveEdit = (id: number) => {
    const e = edits[id]; if (!e) return
    void act(() => apiFetch('/odn/milestones/' + id, { method: 'PUT', body: {
      name: e.name.trim(), plannedDate: e.plannedDate.trim() } }), '里程碑已保存')
  }
  const mark = (id: number, to: 'DONE' | 'PENDING') =>
    void act(() => apiFetch('/odn/milestones/' + id + (to === 'DONE' ? '/complete' : '/reopen'), { method: 'POST' }),
      to === 'DONE' ? '已标记完成' : '已退回未完成')

  const editable = status === 'PENDING'
  const markable = status === 'PENDING' || status === 'BUILDING'
  const budgetAmt = project?.budgetAmount
  const settled = project?.settledAmount ?? 0
  const pct = budgetAmt != null && budgetAmt > 0 ? Math.round((settled / budgetAmt) * 100) : null

  return <div className='space-y-3'>
    <div className={CARD + ' p-4'}>
      <div className='mb-2 text-sm font-semibold'>工程预算<span className='ml-2 text-xs font-normal opacity-60'>预算与里程碑清单仅待开工期可改;执行进度=已结算金额/预算(只读派生,已结算=SETTLED 结算单合计)</span></div>
      <div className='flex flex-wrap items-end gap-2'>
        {editable && <><label className={FIELD}><span className={LABEL}>预算金额</span><Input className='w-40' value={budget} onChange={(e) => setBudget(e.target.value.replace(/[^0-9.]/g, ''))} inputMode='decimal' placeholder='留空=清除' /></label>
        <ToolbarButton primary disabled={busy} onClick={saveBudget}>保存预算</ToolbarButton></>}
        {!editable && <span className='text-xs opacity-60'>预算金额:{budgetAmt != null ? fmtMoney(budgetAmt) : '未登记'}{!markable && '(已竣工锁定)'}</span>}
        {budgetAmt != null && <span className='text-xs'>执行进度:{fmtMoney(settled)} / {fmtMoney(budgetAmt)}{pct != null ? '（' + pct + '%）' : ''}</span>}
      </div>
    </div>
    <div className={CARD + ' p-4'}>
      <div className='mb-2 text-sm font-semibold'>里程碑</div>
      {editable && <div className='mb-3 grid grid-cols-2 gap-3 md:grid-cols-4'>
        <label className={FIELD}><span className={LABEL}>名称</span><Input value={name} onChange={(e) => setName(e.target.value)} placeholder='如 主干光缆敷设完成' /></label>
        <label className={FIELD}><span className={LABEL}>计划完成日</span><Input value={planned} onChange={(e) => setPlanned(e.target.value)} placeholder='YYYY-MM-DD(可空)' /></label>
        <div className='flex items-end'><ToolbarButton primary disabled={busy} onClick={add}>追加里程碑</ToolbarButton></div>
      </div>}
      {error && <div className='mb-2 text-xs text-[var(--color-danger)]'>{error}</div>}
      {rows.length === 0 ? <EmptyState text='暂无里程碑' /> : <div className='overflow-x-auto'><Table>
        <TableHeader><TableRow><TableHead>名称</TableHead><TableHead>计划完成日</TableHead><TableHead>状态</TableHead><TableHead>完成时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map((m) => {
            const e = edits[m.id]
            return <TableRow key={m.id}>
              <TableCell>
                {editable && e
                  ? <Input className='w-48' value={e.name} onChange={(ev) => setEdits((p) => ({ ...p, [m.id]: { name: ev.target.value, plannedDate: e.plannedDate } }))} />
                  : m.name}
              </TableCell>
              <TableCell>
                {editable && e
                  ? <Input className='w-36' value={e.plannedDate} onChange={(ev) => setEdits((p) => ({ ...p, [m.id]: { name: e.name, plannedDate: ev.target.value } }))} placeholder='YYYY-MM-DD' />
                  : (m.plannedDate || '-')}
              </TableCell>
              <TableCell><Badge variant={M_VARIANT[m.status] ?? 'default'}>{M_TEXT[m.status] ?? m.status}</Badge></TableCell>
              <TableCell>{m.doneAt || '-'}</TableCell>
              <TableCell>
                <div className='flex gap-2'>
                  {editable && (e
                    ? <ToolbarButton disabled={busy || !e.name.trim()} onClick={() => saveEdit(m.id)}>保存</ToolbarButton>
                    : <ToolbarButton disabled={busy} onClick={() => setEdits((p) => ({ ...p, [m.id]: { name: m.name, plannedDate: m.plannedDate ?? '' } }))}>编辑</ToolbarButton>)}
                  {markable && (m.status === 'PENDING'
                    ? <ToolbarButton primary disabled={busy} onClick={() => mark(m.id, 'DONE')}>标记完成</ToolbarButton>
                    : <ToolbarButton disabled={busy} onClick={() => mark(m.id, 'PENDING')}>退回</ToolbarButton>)}
                  {!editable && !markable && <span className='text-xs opacity-40'>已锁定</span>}
                </div>
              </TableCell>
            </TableRow>
          })}
        </TableBody>
      </Table></div>}
    </div>
  </div>
}