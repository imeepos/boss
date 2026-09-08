// W1 承包商与工程结算面板(挂 ODN 管理页施工项目页签;P-INFRA-1 W1)。
// 文案为字面量:W1 约束禁触 i18n 中央登记文件(types/locales,W2 才放行)。
// 新建走右侧抽屉 ConstructionCreateDrawer(2026-09-08):与全站表单口径统一,不再用页内内联卡片。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { ConstructionDetail } from './ConstructionDetail'
import { ConstructionCreateDrawer } from './ConstructionCreateDrawer'

export interface Project {
  id: number
  projNo: string
  name: string
  status: string
  itemCount: number
  contractorId: number
  contractorName: string
  itemsAmount: number
  budgetAmount?: number | null
  settledAmount?: number
  asbuiltNote: string
  acceptedAt?: string
  updatedAt: string
}

const STATUS_TEXT: Record<string, string> = { PENDING: '待开工', BUILDING: '施工中', ACCEPTED: '已竣工' }
const STATUS_VARIANT: Record<string, 'default' | 'success' | 'warning'> = { PENDING: 'default', BUILDING: 'warning', ACCEPTED: 'success' }

// fmtMoney 金额展示(两位小数,千分位;金额一律后端计算,前端只展示)。
export function fmtMoney(v: number | null | undefined): string {
  return (v ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// fmtBudgetProgress 预算执行进度展示(已结算/预算,只读派生;未登记禁止显示 0,W6 口径)。
export function fmtBudgetProgress(p: Project): string {
  if (p.budgetAmount == null) return '-'
  const pct = p.budgetAmount > 0 ? Math.round(((p.settledAmount ?? 0) / p.budgetAmount) * 100) : 0
  return fmtMoney(p.settledAmount ?? 0) + ' / ' + fmtMoney(p.budgetAmount) + '（' + pct + '%）'
}

export default function ConstructionsPanel() {
  const [rows, setRows] = useState<Project[]>([])
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [openId, setOpenId] = useState<number | null>(null)

  const load = useCallback(async () => {
    setError('')
    try { setRows((await apiFetch<Project[]>('/odn/constructions', { query: { limit: 100 } })) ?? []) }
    catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [])

  useEffect(() => { void load() }, [load])

  return <div>
    <div className='mb-3 flex items-center justify-between'>
      <ToolbarButton primary onClick={() => setShowCreate(true)}>新建施工单</ToolbarButton>
      <ToolbarButton onClick={() => void load()}>刷新</ToolbarButton>
    </div>
    {error && <ErrorBanner message={error} className="mb-3" />}
    <Card className="overflow-hidden">
      {rows.length === 0 ? <EmptyState text='暂无施工单' /> : <div className='overflow-x-auto'><Table>
        <TableHeader><TableRow><TableHead>施工单号</TableHead><TableHead>名称</TableHead><TableHead>状态</TableHead><TableHead>承包商</TableHead><TableHead>明细数</TableHead><TableHead>清单金额</TableHead><TableHead>预算执行(已结算/预算)</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map((r) => <TableRow key={r.id}>
            <TableCell className='font-mono'>{r.projNo}</TableCell>
            <TableCell>{r.name || '-'}</TableCell>
            <TableCell><Badge variant={STATUS_VARIANT[r.status] ?? 'default'}>{STATUS_TEXT[r.status] ?? r.status}</Badge></TableCell>
            <TableCell>{r.contractorName || <span className='text-xs opacity-60'>未指定</span>}</TableCell>
            <TableCell>{r.itemCount}</TableCell>
            <TableCell>{fmtMoney(r.itemsAmount)}</TableCell>
            <TableCell className='whitespace-nowrap'>{fmtBudgetProgress(r)}</TableCell>
            <TableCell><button className='text-[var(--color-text-link)]' onClick={() => setOpenId(openId === r.id ? null : r.id)}>{openId === r.id ? '收起' : '详情'}</button></TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </Card>
    {openId != null && <ConstructionDetail projectId={openId} onChanged={() => void load()} />}
    {showCreate && <ConstructionCreateDrawer onClose={() => setShowCreate(false)} onCreated={() => void load()} />}
  </div>
}