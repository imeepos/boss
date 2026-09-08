// 预算与里程碑面板(P-INFRA-1 W6/G2,000218;挂施工单详情)。
// 文案为字面量:沿用 ODN 施工页签先例,中央登记仅登记菜单项。
// 追加/编辑里程碑与编辑预算一律右侧抽屉承载(全站抽屉口径),状态按钮保留行内。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { fmtMoney } from './constructions'
import { BudgetEditDrawer } from './BudgetEditDrawer'
import { MilestoneDrawer, type Milestone, type MilestoneDrawerTarget } from './MilestoneDrawer'

const M_VARIANT: Record<string, 'default' | 'success'> = { PENDING: 'default', DONE: 'success' }
const M_TEXT: Record<string, string> = { PENDING: '未完成', DONE: '已完成' }

export function BudgetMilestonePanel({ project, onChanged }: {
  project: { id: number; status: string; budgetAmount?: number | null; settledAmount?: number } | null
  onChanged: () => void
}) {
  const [rows, setRows] = useState<Milestone[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [mTarget, setMTarget] = useState<MilestoneDrawerTarget | null>(null)
  const [budgetOpen, setBudgetOpen] = useState(false)

  const pid = project?.id
  const load = useCallback(async () => {
    if (!pid) return
    try { setRows((await apiFetch<Milestone[]>('/odn/constructions/' + pid + '/milestones')) ?? []) }
    catch (e) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
      toast.error('里程碑加载失败', { description: msg })
    }
  }, [pid])
  useEffect(() => { void load() }, [load])

  const act = async (fn: () => Promise<unknown>, okMsg: string) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await load(); onChanged() }
    catch (e) {
      const msg = e instanceof Error ? e.message : '操作失败'
      setError(msg)
      toast.error(okMsg + ' 失败', { description: msg })
    } finally { setBusy(false) }
  }
  const mark = (id: number, to: 'DONE' | 'PENDING') =>
    void act(() => apiFetch('/odn/milestones/' + id + (to === 'DONE' ? '/complete' : '/reopen'), { method: 'POST' }),
      to === 'DONE' ? '已标记完成' : '已退回未完成')

  const editable = project?.status === 'PENDING'
  const markable = project?.status === 'PENDING' || project?.status === 'BUILDING'
  const budgetAmt = project?.budgetAmount
  const settled = project?.settledAmount ?? 0
  const pct = budgetAmt != null && budgetAmt > 0 ? Math.round((settled / budgetAmt) * 100) : null

  return <div className="space-y-3">
    <Card className="p-4">
      <div className="mb-2 text-sm font-semibold">工程预算<span className="ml-2 text-xs font-normal opacity-60">预算与里程碑清单仅待开工期可改;执行进度=已结算金额/预算(只读派生,已结算=SETTLED 结算单合计)</span></div>
      <div className="flex flex-wrap items-end gap-2">
        {editable && project
          ? <><span className="text-xs">预算金额:{budgetAmt != null ? fmtMoney(budgetAmt) : '未登记'}</span>
            <ToolbarButton disabled={busy} onClick={() => setBudgetOpen(true)}>编辑预算</ToolbarButton></>
          : <span className="text-xs opacity-60">预算金额:{budgetAmt != null ? fmtMoney(budgetAmt) : '未登记'}{!markable && '(已竣工锁定)'}</span>}
        {budgetAmt != null && <span className="text-xs">执行进度:{fmtMoney(settled)} / {fmtMoney(budgetAmt)}{pct != null ? '（' + pct + '%）' : ''}</span>}
      </div>
    </Card>
    <Card className="p-4">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-sm font-semibold">里程碑</span>
        {editable && project && <ToolbarButton primary disabled={busy} onClick={() => setMTarget({ projectId: project.id, editing: null })}>追加里程碑</ToolbarButton>}
      </div>
      {error && <ErrorBanner message={error} className="mb-2" />}
      {rows.length === 0 ? <EmptyState text="暂无里程碑" /> : <div className="overflow-x-auto"><Table>
        <TableHeader><TableRow><TableHead>名称</TableHead><TableHead>计划完成日</TableHead><TableHead>状态</TableHead><TableHead>完成时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map((m) => <TableRow key={m.id}>
            <TableCell>{m.name}</TableCell>
            <TableCell>{m.plannedDate || '-'}</TableCell>
            <TableCell><Badge variant={M_VARIANT[m.status] ?? 'default'}>{M_TEXT[m.status] ?? m.status}</Badge></TableCell>
            <TableCell>{m.doneAt || '-'}</TableCell>
            <TableCell>
              <div className='flex gap-2'>
                {editable && project && <ToolbarButton disabled={busy} onClick={() => setMTarget({ projectId: project.id, editing: m })}>编辑</ToolbarButton>}
                {markable && (m.status === 'PENDING'
                  ? <ToolbarButton primary disabled={busy} onClick={() => mark(m.id, 'DONE')}>标记完成</ToolbarButton>
                  : <ToolbarButton disabled={busy} onClick={() => mark(m.id, 'PENDING')}>退回</ToolbarButton>)}
                {!editable && !markable && <span className='text-xs opacity-40'>已锁定</span>}
              </div>
            </TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </Card>
    {mTarget && <MilestoneDrawer target={mTarget} onClose={() => setMTarget(null)}
      onSaved={() => { void load(); onChanged() }} />}
    {budgetOpen && project && <BudgetEditDrawer projectId={project.id} current={project.budgetAmount}
      onClose={() => setBudgetOpen(false)} onSaved={() => { void load(); onChanged() }} />}
  </div>
}
