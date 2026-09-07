// W1 承包商与工程结算面板(挂 ODN 管理页施工项目页签;P-INFRA-1 W1)。
// 文案为字面量:W1 约束禁触 i18n 中央登记文件(types/locales,W2 才放行)。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { ConstructionDetail } from './ConstructionDetail'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

export interface Project {
  id: number
  projNo: string
  name: string
  status: string
  itemCount: number
  contractorId: number
  contractorName: string
  itemsAmount: number
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

export default function ConstructionsPanel() {
  const [rows, setRows] = useState<Project[]>([])
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [projNo, setProjNo] = useState('')
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [openId, setOpenId] = useState<number | null>(null)

  const load = useCallback(async () => {
    setError('')
    try { setRows((await apiFetch<Project[]>('/odn/constructions', { query: { limit: 100 } })) ?? []) }
    catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [])

  useEffect(() => { void load() }, [load])

  const create = async () => {
    if (!projNo.trim()) { setError('施工单号必填'); return }
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/constructions', { method: 'POST', body: { projNo: projNo.trim(), name: name.trim() } })
      toast.success('施工单已创建')
      setProjNo(''); setName(''); setShowForm(false)
      await load()
    } catch (e) { setError(e instanceof Error ? e.message : '保存失败') } finally { setBusy(false) }
  }

  return <div>
    <div className='mb-3 flex items-center justify-between'>
      <ToolbarButton primary onClick={() => setShowForm(!showForm)}>{showForm ? '取消' : '新建施工单'}</ToolbarButton>
      <ToolbarButton onClick={() => void load()}>刷新</ToolbarButton>
    </div>
    {showForm && <div className={CARD + ' mb-3 p-4'}>
      <div className='grid grid-cols-2 gap-3 md:grid-cols-4'>
        <label className={FIELD}><span className={LABEL}>施工单号</span><Input value={projNo} onChange={(e) => setProjNo(e.target.value)} placeholder='C-20260907-001' /></label>
        <label className={FIELD}><span className={LABEL}>名称</span><Input value={name} onChange={(e) => setName(e.target.value)} /></label>
      </div>
      <div className='mt-3 flex justify-end'><ToolbarButton primary disabled={busy} onClick={() => void create()}>{busy ? '保存中…' : '保存'}</ToolbarButton></div>
    </div>}
    {error && <ErrorBanner message={error} className='mb-3' />}
    <section className={CARD + ' overflow-hidden'}>
      {rows.length === 0 ? <EmptyState text='暂无施工单' /> : <div className='overflow-x-auto'><Table>
        <TableHeader><TableRow><TableHead>施工单号</TableHead><TableHead>名称</TableHead><TableHead>状态</TableHead><TableHead>承包商</TableHead><TableHead>明细数</TableHead><TableHead>清单金额</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map((r) => <TableRow key={r.id}>
            <TableCell className='font-mono'>{r.projNo}</TableCell>
            <TableCell>{r.name || '-'}</TableCell>
            <TableCell><Badge variant={STATUS_VARIANT[r.status] ?? 'default'}>{STATUS_TEXT[r.status] ?? r.status}</Badge></TableCell>
            <TableCell>{r.contractorName || <span className='text-xs opacity-60'>未指定</span>}</TableCell>
            <TableCell>{r.itemCount}</TableCell>
            <TableCell>{fmtMoney(r.itemsAmount)}</TableCell>
            <TableCell><button className='text-[var(--color-text-link)]' onClick={() => setOpenId(openId === r.id ? null : r.id)}>{openId === r.id ? '收起' : '详情'}</button></TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </section>
    {openId != null && <ConstructionDetail projectId={openId} onChanged={() => void load()} />}
  </div>
}