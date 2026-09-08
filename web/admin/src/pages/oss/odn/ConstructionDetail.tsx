// 施工单详情:承包商指定 / 工程量清单编辑 / 结算发起与列表(P-INFRA-1 W1)。
// 设施关联走 pickers 选择器(2026-09-07 域改造);新增文案走 pages.odn 三语词条。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { useT } from '../../../i18n'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'
import { fmtMoney, type Project } from './constructions'
import { BudgetMilestonePanel } from './BudgetMilestonePanel'
import { PERMIT_KIND_TEXT, PERMIT_STATUS_TEXT, PERMIT_STATUS_VARIANT } from './permits'
import { MaterialIssuesCard } from './MaterialIssuesCard'
import { ProgressCard } from './ProgressCard'

interface Item { id: number; projectId: number; facilityCode: string; quantity: number; unitPrice: number; amount: number }
interface PermitLite { id: number; permitNo: string; kind: string; status: string; validUntil?: string }
interface Settlement {
  id: number; settlementNo: string; projectId: number; projectNo: string
  contractorId: number; contractorName: string; totalAmount: number; itemCount: number
  status: string; voidReason: string; createdAt: string; settledAt?: string; voidedAt?: string
}
interface Supplier { id: number; code: string; name: string; contractorType: string; status: string }
// FacilityLite 设施选择器数据源行(/odn/facilities 全网 ≤500,字段以 internal/domain/odn 为准)。
interface FacilityLite { code: string; name: string }
const S_TEXT: Record<string, string> = { PENDING: '待结算', SETTLED: '已结算', VOIDED: '已作废' }
const S_VARIANT: Record<string, 'info' | 'success' | 'danger'> = { PENDING: 'info', SETTLED: 'success', VOIDED: 'danger' }

export function ConstructionDetail({ projectId, onChanged }: { projectId: number; onChanged: () => void }) {
  const confirmDialog = useConfirm()
  const t = useT()
  const [project, setProject] = useState<Project | null>(null)
  const [items, setItems] = useState<Item[]>([])
  const [settlements, setSettlements] = useState<Settlement[]>([])
  const [permits, setPermits] = useState<PermitLite[]>([])
  const [unlinked, setUnlinked] = useState<PermitLite[]>([])
  const [linkId, setLinkId] = useState('')
  const [suppliers, setSuppliers] = useState<Supplier[]>([])
  const [contractorId, setContractorId] = useState('')
  const [fac, setFac] = useState(''); const [qty, setQty] = useState(''); const [price, setPrice] = useState('')
  const [edits, setEdits] = useState<Record<number, { quantity: string; unitPrice: string }>>({})
  const [voidReason, setVoidReason] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [facOpts, setFacOpts] = useState<DropdownOption[]>([])

  const load = useCallback(async () => {
    setError('')
    try {
      const d = await apiFetch<{ project: Project; items: Item[] }>('/odn/constructions/' + projectId)
      setProject(d?.project ?? null); setItems(d?.items ?? [])
      setSettlements((await apiFetch<Settlement[]>('/odn/constructions/' + projectId + '/settlements')) ?? [])
      setPermits((await apiFetch<PermitLite[]>('/odn/constructions/' + projectId + '/permits')) ?? [])
      setUnlinked((await apiFetch<PermitLite[]>('/odn/permits', { query: { unlinked: 1, limit: 200 } })) ?? [])
    } catch (e) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
      toast.error('施工单加载失败', { description: msg })
    }
  }, [projectId])

  useEffect(() => { void load() }, [load])
  useEffect(() => {
    void (async () => {
      try {
        const all = (await apiFetch<Supplier[]>('/procurement/suppliers', { query: { limit: 500 } })) ?? []
        setSuppliers(all.filter((s) => s.contractorType === 'CONSTRUCTION' && s.status === 'ENABLED'))
      } catch { setSuppliers([]) }
    })()
  }, [])
  // 设施主数据静态源(工程量清单选择器;coverage 页签同口径,空参=全网)。
  useEffect(() => {
    void (async () => {
      try {
        const facs = (await apiFetch<FacilityLite[]>('/odn/facilities')) ?? []
        setFacOpts(facs.map((x) => ({ value: x.code, label: x.code + (x.name ? ' ' + x.name : '') })))
      } catch { setFacOpts([]) }
    })()
  }, [])

  const act = async (fn: () => Promise<unknown>, okMsg: string) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await load(); onChanged() }
    catch (e) {
      const msg = e instanceof Error ? e.message : '操作失败'
      setError(msg)
      toast.error(okMsg + ' 失败', { description: msg })
    } finally { setBusy(false) }
  }

  const assignContractor = () => act(() => apiFetch('/odn/constructions/' + projectId + '/contractor',
    { method: 'PUT', body: { contractorId: Number(contractorId) } }), '承包商已指定')
  const addItem = () => {
    if (!fac.trim()) { setError('设施编码必填'); return }
    void act(() => apiFetch('/odn/constructions/' + projectId + '/items', { method: 'POST', body: {
      facilityCode: fac.trim(), quantity: Number(qty) || 0, unitPrice: Number(price) || 0 } }), '明细已追加')
      .then(() => { setFac(''); setQty(''); setPrice('') })
  }
  const saveItem = (id: number) => {
    const e = edits[id]; if (!e) return
    void act(() => apiFetch('/odn/constructions/' + projectId + '/items/' + id, { method: 'PUT', body: {
      quantity: Number(e.quantity) || 0, unitPrice: Number(e.unitPrice) || 0 } }), '定额已保存')
  }
  const start = () => act(() => apiFetch('/odn/constructions/' + projectId + '/start', { method: 'POST' }), '已开工')
  const linkPermit = () => {
    if (!linkId) { setError('选择要关联的许可单'); return }
    void act(() => apiFetch('/odn/permits/' + linkId + '/link', { method: 'POST', body: { projectId } }), '许可已关联')
      .then(() => setLinkId(''))
  }
  const unlinkPermit = (id: number) => void act(() => apiFetch('/odn/permits/' + id + '/unlink', { method: 'POST' }), '许可已解除关联')
  const accept = async () => {
    if (!(await confirmDialog('确认竣工验收?竣工后明细与清单锁定,单内设施回填在服。', { danger: true }))) return
    void act(() => apiFetch('/odn/constructions/' + projectId + '/accept', { method: 'POST', body: { note: '' } }), '已竣工')
  }
  const settle = () => { if (!active) return; void act(() => apiFetch('/odn/settlements/' + active.id + '/settle', { method: 'POST' }), '结算已确认') }
  const voidSettlement = async () => {
    if (!voidReason.trim()) { setError('作废必填原因'); return }
    if (!(await confirmDialog('确认作废该结算单?作废后可重新发起。', { danger: true }))) return
    if (!active) return
    await act(() => apiFetch('/odn/settlements/' + active.id + '/void', { method: 'POST', body: { reason: voidReason.trim() } }), '结算单已作废')
    setVoidReason('')
  }
  const createSettlement = async () => {
    if (!(await confirmDialog('按清单金额汇总发起结算?', {}))) return
    await act(() => apiFetch('/odn/constructions/' + projectId + '/settlements', { method: 'POST' }), '结算已发起')
  }

  const lockSettle = settlements.some((s) => s.status !== 'VOIDED')
  const active = settlements.find((s) => s.status !== 'VOIDED')

  return <div className="mt-4 space-y-3">
    {error && <ErrorBanner message={error} />}
    <BudgetMilestonePanel project={project} onChanged={onChanged} />
    <ProgressCard projectId={projectId} />
    <Card className="p-4">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold">承包商</span>
        {project && project.contractorName && <Badge>{project.contractorName}</Badge>}
        {project && !project.contractorName && <span className="text-xs opacity-60">未指定(存量项目兼容;结算前须指定施工类供应商)</span>}
        {!lockSettle && <div className="flex items-end gap-2">
          <Dropdown value={contractorId} ariaLabel="选择施工类供应商" placeholder="选择施工类供应商" searchable searchPlaceholder="搜索供应商"
            options={suppliers.map((s) => ({ value: String(s.id), label: s.name + ' (' + s.code + ')' }))}
            onChange={(v) => setContractorId(v)} />
          <ToolbarButton primary disabled={busy || !contractorId} onClick={() => void assignContractor()}>指定</ToolbarButton>
        </div>}
        {lockSettle && <span className="text-xs opacity-60">已有有效结算单,承包商锁定</span>}
      </div>
      <div className="flex flex-wrap gap-2">
        {project?.status === 'PENDING' && <ToolbarButton primary disabled={busy} onClick={() => void start()}>开工</ToolbarButton>}
        {project?.status === 'BUILDING' && <ToolbarButton primary disabled={busy} onClick={() => void accept()}>竣工验收</ToolbarButton>}
        {project?.status === 'ACCEPTED' && !lockSettle &&
          <ToolbarButton primary disabled={busy || !project.contractorId} onClick={() => void createSettlement()}>发起结算</ToolbarButton>}
        {project?.status === 'ACCEPTED' && !project.contractorId && <span className="text-xs opacity-60">竣工未指定承包商,不可发起结算</span>}
      </div>
    </Card>

    <Card className="p-4">
      <div className="mb-2 text-sm font-semibold">工程量清单{project?.status === 'ACCEPTED' && <span className="ml-2 text-xs opacity-60">已竣工锁定</span>}</div>
      {project && project.status !== 'ACCEPTED' && <div className="mb-3 grid grid-cols-2 gap-3 md:grid-cols-4">
        <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">设施编码</span><SimplePicker value={fac} onChange={setFac} options={facOpts} ariaLabel={t.pages.odn.pickFacility} searchPlaceholder={t.pages.odn.pickFacilitySearch} clearable clearLabel={t.pages.pickers.common.clear} minWidth={200} /></label>
        <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">数量</span><Input value={qty} onChange={(e) => setQty(e.target.value)} inputMode="decimal" /></label>
        <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">单价</span><Input value={price} onChange={(e) => setPrice(e.target.value)} inputMode="decimal" /></label>
        <div className="flex items-end"><ToolbarButton primary disabled={busy} onClick={addItem}>追加明细</ToolbarButton></div>
      </div>}
      {items.length === 0 ? <EmptyState text='暂无明细' /> : <div className='overflow-x-auto'><Table>
        <TableHeader><TableRow><TableHead>设施</TableHead><TableHead>数量</TableHead><TableHead>单价</TableHead><TableHead>金额(后端计算)</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {items.map((it) => {
            const e = edits[it.id]
            const editable = project != null && project.status !== 'ACCEPTED'
            return <TableRow key={it.id}>
              <TableCell className='font-mono'>{it.facilityCode}</TableCell>
              {editable ? <>
                <TableCell><Input className='w-24' value={e ? e.quantity : String(it.quantity)} onChange={(ev) => setEdits((m) => ({ ...m, [it.id]: { quantity: ev.target.value, unitPrice: e ? e.unitPrice : String(it.unitPrice) } }))} inputMode='decimal' /></TableCell>
                <TableCell><Input className='w-24' value={e ? e.unitPrice : String(it.unitPrice)} onChange={(ev) => setEdits((m) => ({ ...m, [it.id]: { quantity: e ? e.quantity : String(it.quantity), unitPrice: ev.target.value } }))} inputMode='decimal' /></TableCell>
                <TableCell>{fmtMoney(it.amount)}</TableCell>
                <TableCell><ToolbarButton disabled={busy || !e} onClick={() => saveItem(it.id)}>保存定额</ToolbarButton></TableCell>
              </> : <>
                <TableCell>{it.quantity}</TableCell><TableCell>{it.unitPrice}</TableCell>
                <TableCell>{fmtMoney(it.amount)}</TableCell><TableCell>-</TableCell>
              </>}
            </TableRow>
          })}
        </TableBody>
      </Table></div>}
    </Card>

    <Card className="p-4">
      <div className="mb-2 text-sm font-semibold">开工许可<span className="ml-2 text-xs font-normal opacity-60">许可前置门控开启时,未获批/未盖章的许可将拒绝开工(P-INFRA-1 W4)</span></div>
      <div className="mb-3 flex flex-wrap items-end gap-2">
        <Dropdown value={linkId} ariaLabel="选择待关联许可" placeholder="选择未关联许可单" searchable searchPlaceholder="搜索单号"
          options={unlinked.map((p) => ({ value: String(p.id), label: p.permitNo + ' (' + (PERMIT_KIND_TEXT[p.kind] ?? p.kind) + ')' }))}
          onChange={(v) => setLinkId(v)} />
        <ToolbarButton primary disabled={busy || !linkId} onClick={linkPermit}>关联许可</ToolbarButton>
      </div>
      {permits.length === 0 ? <EmptyState text="暂无关联许可(覆盖门控开启时开工将被拒绝)" /> : <div className="overflow-x-auto"><Table>
        <TableHeader><TableRow><TableHead>许可单号</TableHead><TableHead>类型</TableHead><TableHead>状态</TableHead><TableHead>有效期止</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {permits.map((p) => <TableRow key={p.id}>
            <TableCell className='font-mono'>{p.permitNo}</TableCell>
            <TableCell>{PERMIT_KIND_TEXT[p.kind] ?? p.kind}</TableCell>
            <TableCell><Badge variant={PERMIT_STATUS_VARIANT[p.status] ?? 'default'}>{PERMIT_STATUS_TEXT[p.status] ?? p.status}</Badge></TableCell>
            <TableCell>{p.validUntil || '-'}</TableCell>
            <TableCell><ToolbarButton disabled={busy} onClick={() => unlinkPermit(p.id)}>解除关联</ToolbarButton></TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </Card>

    <Card className="p-4">
      <div className="mb-2 text-sm font-semibold">工程结算</div>
      {active?.status === 'PENDING' && <div className="mb-2 flex flex-wrap items-end gap-2">
        <ToolbarButton primary disabled={busy} onClick={() => void settle()}>确认结算</ToolbarButton>
        <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">作废原因</span><Input value={voidReason} onChange={(e) => setVoidReason(e.target.value)} placeholder="作废必填原因" /></label>
        <ToolbarButton disabled={busy} onClick={() => void voidSettlement()}>作废</ToolbarButton>
      </div>}
      {settlements.length === 0 ? <EmptyState text="暂无结算单" /> : <div className="overflow-x-auto"><Table>
        <TableHeader><TableRow><TableHead>结算单号</TableHead><TableHead>承包商</TableHead><TableHead>应付金额</TableHead><TableHead>明细数</TableHead><TableHead>状态</TableHead><TableHead>作废原因</TableHead><TableHead>发起时间</TableHead></TableRow></TableHeader>
        <TableBody>
          {settlements.map((s) => <TableRow key={s.id}>
            <TableCell className='font-mono'>{s.settlementNo}</TableCell>
            <TableCell>{s.contractorName || '-'}</TableCell>
            <TableCell>{fmtMoney(s.totalAmount)}</TableCell>
            <TableCell>{s.itemCount}</TableCell>
            <TableCell><Badge variant={S_VARIANT[s.status] ?? 'default'}>{S_TEXT[s.status] ?? s.status}</Badge></TableCell>
            <TableCell>{s.voidReason || '-'}</TableCell>
            <TableCell>{s.createdAt}</TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </Card>

    <MaterialIssuesCard projectId={projectId} locked={project?.status === 'ACCEPTED'} />,
  </div>
}