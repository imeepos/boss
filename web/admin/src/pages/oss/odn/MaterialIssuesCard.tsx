// 材料出库卡片:自 ConstructionDetail 等价拆出(纯结构迁移,行为不变)。
// 出库至本项目工地的资产台账连续可查;材料成本归集归 W9 项目领料。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

interface MaterialIssue {
  id: number; issueNo: string; projectId: number; projectNo: string; status: string
  remark: string; assetIds: number[]; createdBy: number; issuedBy: number; cancelledBy: number
  createdAt: string; issuedAt?: string; cancelledAt?: string
}

const ISSUE_TEXT: Record<string, string> = { OPEN: '备出库', CONFIRMED: '已出库(在途)', CANCELLED: '已取消' }
const ISSUE_VARIANT: Record<string, 'info' | 'success' | 'danger'> = { OPEN: 'info', CONFIRMED: 'success', CANCELLED: 'danger' }

export function MaterialIssuesCard({ projectId, locked }: { projectId: number; locked: boolean }) {
  const confirmDialog = useConfirm()
  const [issues, setIssues] = useState<MaterialIssue[]>([])
  const [issueAssets, setIssueAssets] = useState('')
  const [issueRemark, setIssueRemark] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    try { setIssues((await apiFetch<MaterialIssue[]>('/odn/material-issues', { query: { projectId } })) ?? []) }
    catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [projectId])
  useEffect(() => { void load() }, [load])

  const act = async (fn: () => Promise<unknown>, okMsg: string) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await load() }
    catch (e) { setError(e instanceof Error ? e.message : '操作失败') } finally { setBusy(false) }
  }

  const createIssue = () => {
    const ids = issueAssets.split(/[,，\s]+/).map((s) => Number(s.trim())).filter((n) => Number.isFinite(n) && n > 0)
    if (ids.length === 0) { setError('填写要出库的资产 ID(IN_STOCK)'); return }
    void act(() => apiFetch('/odn/material-issues', { method: 'POST', body: {
      projectId, assetIds: ids, remark: issueRemark.trim() } }), '出库单已创建').then(() => { setIssueAssets(''); setIssueRemark('') })
  }
  const confirmIssue = (id: number) => void act(() => apiFetch('/odn/material-issues/' + id + '/confirm', { method: 'POST' }), '已出库,资产转在途')
  const cancelIssue = async (id: number) => {
    if (!(await confirmDialog('取消该出库单?已出库的资产将退库回 IN_STOCK。', { danger: true }))) return
    void act(() => apiFetch('/odn/material-issues/' + id + '/cancel', { method: 'POST' }), '出库单已取消')
  }

  return <div className={CARD + ' p-4'}>
    <div className='mb-2 text-sm font-semibold'>材料出库<span className='ml-2 text-xs font-normal opacity-60'>出库至本项目工地的资产台账连续可查(在途 IN_TRANSIT,转固后 DEPLOYED);材料成本归集归 W9 项目领料</span></div>
    {!locked && <div className='mb-3 grid grid-cols-2 gap-3 md:grid-cols-4'>
      <label className={FIELD}><span className={LABEL}>资产 ID(逗号分隔,须 IN_STOCK)</span><Input value={issueAssets} onChange={(e) => setIssueAssets(e.target.value)} placeholder='3001,3002,3003' /></label>
      <label className={FIELD}><span className={LABEL}>备注</span><Input value={issueRemark} onChange={(e) => setIssueRemark(e.target.value)} placeholder='可空' /></label>
      <div className='flex items-end'><ToolbarButton primary disabled={busy} onClick={createIssue}>创建出库单</ToolbarButton></div>
    </div>}
    {error && <ErrorBanner message={error} className='mb-2' />}
    {issues.length === 0 ? <EmptyState text='暂无出库单' /> : <div className='overflow-x-auto'><Table>
      <TableHeader><TableRow><TableHead>出库单号</TableHead><TableHead>资产</TableHead><TableHead>状态</TableHead><TableHead>备注</TableHead><TableHead>创建时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
      <TableBody>
        {issues.map((m) => <TableRow key={m.id}>
          <TableCell className='font-mono'>{m.issueNo}</TableCell>
          <TableCell className='font-mono'>{m.assetIds.join(', ')}</TableCell>
          <TableCell><Badge variant={ISSUE_VARIANT[m.status] ?? 'default'}>{ISSUE_TEXT[m.status] ?? m.status}</Badge></TableCell>
          <TableCell>{m.remark || '-'}</TableCell>
          <TableCell>{m.createdAt}</TableCell>
          <TableCell>{(m.status === 'OPEN' || m.status === 'CONFIRMED')
            ? <div className='flex gap-2'>
              {m.status === 'OPEN' && <ToolbarButton primary disabled={busy} onClick={() => confirmIssue(m.id)}>确认出库</ToolbarButton>}
              <ToolbarButton disabled={busy} onClick={() => void cancelIssue(m.id)}>{m.status === 'CONFIRMED' ? '退库取消' : '取消'}</ToolbarButton>
            </div> : <span className='text-xs opacity-40'>-</span>}</TableCell>
        </TableRow>)}
      </TableBody>
    </Table></div>}
  </div>
}