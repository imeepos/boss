// 材料出库卡片:资产选择走 DialogPicker 多选(分页+关键字),提交结构 assetIds 不变。
// 出库至本项目工地的资产台账连续可查;材料成本归集归 W9 项目领料。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'
import { DialogPicker, type DialogPickerQuery, type DialogPickerPage } from '../../../components/pickers/DialogPicker'
import { PickerChips } from '../../../components/pickers/DialogPickerParts'
import { useT } from '../../../i18n'

interface MaterialIssue {
  id: number; issueNo: string; projectId: number; projectNo: string; status: string
  remark: string; assetIds: number[]; createdBy: number; issuedBy: number; cancelledBy: number
  createdAt: string; issuedAt?: string; cancelledAt?: string
}

// AssetPick 出库候选资产行(对齐 /assets 分页接口,字段以 internal/domain/asset 为准)。
interface AssetPick { assetId: number; assetCode: string; type: string; status: string }

const ISSUE_TEXT: Record<string, string> = { OPEN: '备出库', CONFIRMED: '已出库(在途)', CANCELLED: '已取消' }
const ISSUE_VARIANT: Record<string, 'info' | 'success' | 'danger'> = { OPEN: 'info', CONFIRMED: 'success', CANCELLED: 'danger' }

export function MaterialIssuesCard({ projectId, locked }: { projectId: number; locked: boolean }) {
  const confirmDialog = useConfirm()
  const t = useT()
  const o = t.pages.odn
  const dlg = t.pages.pickers.dialog
  const [issues, setIssues] = useState<MaterialIssue[]>([])
  const [issueRemark, setIssueRemark] = useState('')
  const [picked, setPicked] = useState<AssetPick[]>([])
  const [pickOpen, setPickOpen] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    try { setIssues((await apiFetch<MaterialIssue[]>('/odn/material-issues', { query: { projectId } })) ?? []) }
    catch (e) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
      toast.error('出库单加载失败', { description: msg })
    }
  }, [projectId])
  useEffect(() => { void load() }, [load])

  const act = async (fn: () => Promise<unknown>, okMsg: string) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await load() }
    catch (e) {
      const msg = e instanceof Error ? e.message : '操作失败'
      setError(msg)
      toast.error(okMsg + ' 失败', { description: msg })
    } finally { setBusy(false) }
  }

  const createIssue = () => {
    if (picked.length === 0) { setError(o.issueNeedAsset); return }
    void act(() => apiFetch('/odn/material-issues', { method: 'POST', body: {
      projectId, assetIds: picked.map((a) => a.assetId), remark: issueRemark.trim() } }), '出库单已创建')
      .then(() => { setPicked([]); setIssueRemark('') })
  }
  const confirmIssue = (id: number) => void act(() => apiFetch('/odn/material-issues/' + id + '/confirm', { method: 'POST' }), '已出库,资产转在途')
  const cancelIssue = async (id: number) => {
    if (!(await confirmDialog('取消该出库单?已出库的资产将退库回 IN_STOCK。', { danger: true }))) return
    void act(() => apiFetch('/odn/material-issues/' + id + '/cancel', { method: 'POST' }), '出库单已取消')
  }

  // 资产分页查询:q 关键字 + offset/limit 分页(DIST /assets 契约,钳制不报错)。
  const queryAssets = async (q: DialogPickerQuery): Promise<DialogPickerPage<AssetPick>> => {
    const d = await apiFetch<{ items: AssetPick[]; total: number }>('/assets', { query: {
      q: q.keyword || undefined, offset: (q.page - 1) * q.pageSize, limit: q.pageSize } })
    return { items: d?.items ?? [], total: d?.total ?? 0 }
  }

  const assetLabel = (a: AssetPick) => a.assetCode + ' #' + a.assetId
  const chips = picked.map((a) => ({ key: String(a.assetId), label: assetLabel(a) }))

  return <Card className="p-4">
    <div className="mb-2 text-sm font-semibold">材料出库<span className="ml-2 text-xs font-normal opacity-60">出库至本项目工地的资产台账连续可查(在途 IN_TRANSIT,转固后 DEPLOYED);材料成本归集归 W9 项目领料</span></div>
    {!locked && <div className="mb-3 flex flex-wrap items-end gap-3">
      <div className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">{o.issueField}</span>
        <ToolbarButton disabled={busy} onClick={() => setPickOpen(true)}>{o.issuePickBtn}</ToolbarButton>
      </div>
      <PickerChips chips={chips} selectedCount={dlg.selectedCount} removeLabel={dlg.remove} clearAllLabel={dlg.clearAll}
        onRemove={(k) => setPicked((p) => p.filter((x) => String(x.assetId) !== k))} onClearAll={() => setPicked([])} />
      <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">备注</span><Input value={issueRemark} onChange={(e) => setIssueRemark(e.target.value)} placeholder="可空" /></label>
      <div className="flex items-end"><ToolbarButton primary disabled={busy || picked.length === 0} onClick={createIssue}>创建出库单</ToolbarButton></div>
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
    <DialogPicker<AssetPick> open={pickOpen} mode='multiple' title={o.issuePickTitle}
      onClose={() => setPickOpen(false)} onPick={(items) => setPicked(items)}
      columns={[{ key: 'assetCode', title: o.issueColCode }, { key: 'type', title: o.issueColType }, { key: 'status', title: o.issueColStatus }]}
      query={queryAssets} rowKey={(a) => String(a.assetId)} rowLabel={assetLabel} texts={dlg} />
  </Card>
}