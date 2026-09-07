// ODN 资产化转固面板(P-INFRA-1 W8):凭证列表/登记/冲销 + 出库台账入口。
// 关联实体一律 pickers 选择器(2026-09-07 域改造);新增文案走 pages.odn 三语词条。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Dropdown, type DropdownOption } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { useT } from '../../../i18n'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

interface Registration {
  id: number; registrationNo: string; entityKind: string; facilityCode: string; deviceId: number
  assetId: number; sourceKind: string; constructionProjectId: number; batchId: number
  valueAmount: number; status: string; reverseReason: string; remark: string
  registeredBy: number; registeredAt: string
}

const SOURCE_TEXT: Record<string, string> = { PROCUREMENT: '采购入库', CONSTRUCTION: '施工建成', DIRECT: '直购直转' }

// 选择器数据源行类型(本域列表接口;字段以 internal/domain/odn 与 asset 为准)。
interface DeviceLite { id: number; code: string; name: string; status: string }
interface AssetLite { assetId: number; assetCode: string; status: string; type: string }
interface ProjectLite { id: number; projNo: string; name: string; status: string }
const STATUS_VARIANT: Record<string, 'info' | 'success' | 'danger'> = { ACTIVE: 'success', REVERSED: 'danger' }
const STATUS_TEXT: Record<string, string> = { ACTIVE: '有效', REVERSED: '已冲销' }

function entityLabel(r: Registration): string {
  return r.entityKind === 'FACILITY' ? (r.facilityCode || '-') : '设备#' + r.deviceId
}

export function AssetsPanel() {
  const confirmDialog = useConfirm()
  const o = useT().pages.odn
  const [rows, setRows] = useState<Registration[]>([])
  const [status, setStatus] = useState('ACTIVE')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [showForm, setShowForm] = useState(false)
  const [entityKind, setEntityKind] = useState('FACILITY')
  const [facilityCode, setFacilityCode] = useState('')
  const [deviceId, setDeviceId] = useState('')
  const [assetId, setAssetId] = useState('')
  const [sourceKind, setSourceKind] = useState('DIRECT')
  const [projectId, setProjectId] = useState('')
  const [valueAmount, setValueAmount] = useState('')
  const [remark, setRemark] = useState('')
  const [reverseId, setReverseId] = useState('')
  const [reverseReason, setReverseReason] = useState('')
  const [deviceOpts, setDeviceOpts] = useState<DropdownOption[]>([])
  const [projectOpts, setProjectOpts] = useState<DropdownOption[]>([])

  // 设备/施工项目静态源一次拉取(全网 ≤500/100 条,本地过滤;coverage 页签同口径)。
  useEffect(() => {
    void (async () => {
      try {
        const devs = (await apiFetch<DeviceLite[]>('/odn/devices')) ?? []
        setDeviceOpts(devs.map((d) => ({ value: String(d.id), label: (d.name || d.code) + ' (' + d.code + ')' })))
      } catch { setDeviceOpts([]) }
      try {
        const projs = (await apiFetch<ProjectLite[]>('/odn/constructions', { query: { limit: 100 } })) ?? []
        setProjectOpts(projs.map((p) => ({ value: String(p.id), label: p.projNo + ' ' + (p.name || '') + ' [' + (o.projStatus[p.status as keyof typeof o.projStatus] ?? p.status) + ']' })))
      } catch { setProjectOpts([]) }
    })()
  }, [])

  // 资产服务端检索(/assets 分页接口 q 关键字;label 标注状态便于识别可登记状态)。
  const searchAssets = async (kw: string): Promise<DropdownOption[] | null> => {
    const d = await apiFetch<{ items: AssetLite[] }>('/assets', { query: { q: kw || undefined, limit: 50 } })
    return (d?.items ?? []).map((a) => ({ value: String(a.assetId), label: a.assetCode + ' #' + a.assetId + ' [' + a.status + ']' }))
  }

  const load = useCallback(async () => {
    setError('')
    try {
      const q: Record<string, string | number | undefined> = { limit: 200 }
      if (status) q.status = status
      setRows((await apiFetch<Registration[]>('/odn/assets/registrations', { query: q })) ?? [])
    } catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [status])
  useEffect(() => { void load() }, [load])

  const register = async () => {
    const body: Record<string, unknown> = { entityKind, sourceKind, assetId: Number(assetId) || 0 }
    if (entityKind === 'FACILITY') body.facilityCode = facilityCode.trim(); else body.deviceId = Number(deviceId) || 0
    if (sourceKind === 'CONSTRUCTION') body.projectId = Number(projectId) || 0
    if (valueAmount !== '') body.valueAmount = Number(valueAmount) || 0
    if (remark.trim()) body.remark = remark.trim()
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/assets/registrations', { method: 'POST', body })
      toast.success('已登记资产化凭证,资产转为 DEPLOYED')
      setShowForm(false); setFacilityCode(''); setDeviceId(''); setAssetId(''); setProjectId(''); setValueAmount(''); setRemark('')
      await load()
    } catch (e) { setError(e instanceof Error ? e.message : '登记失败') } finally { setBusy(false) }
  }

  const reverse = async (id: number) => {
    if (!reverseReason.trim() || Number(reverseId) !== id) { setError('冲销须填写原因并对应凭证'); return }
    if (!(await confirmDialog('确认冲销该凭证?冲销后资产回 IN_STOCK,凭证保留历史。', { danger: true }))) return
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/assets/registrations/' + id + '/reverse', { method: 'POST', body: { reason: reverseReason.trim() } })
      toast.success('凭证已冲销')
      setReverseId(''); setReverseReason('')
      await load()
    } catch (e) { setError(e instanceof Error ? e.message : '冲销失败') } finally { setBusy(false) }
  }

  const formOk = assetId !== '' && (entityKind === 'FACILITY' ? facilityCode.trim() !== '' : deviceId !== '')
  return <section className={CARD + ' mt-4 p-4'}>
    {error && <ErrorBanner message={error} className='mb-3' />}
    <div className='mb-3 flex flex-wrap items-center gap-2'>
      <span className='text-sm font-semibold'>资产化凭证</span>
      <span className='text-xs opacity-60'>施工建成设施/设备(含导入域箱体)凭证据此获得资产身份;价值与采购/项目溯源随凭证登记</span>
      <div className='ml-auto flex items-end gap-2'>
        <Dropdown value={status} ariaLabel='凭证状态' options={[{ value: 'ACTIVE', label: '有效' }, { value: 'REVERSED', label: '已冲销' }, { value: '', label: '全部' }]} onChange={setStatus} />
        <ToolbarButton primary onClick={() => setShowForm(!showForm)}>{showForm ? '收起' : '资产化登记'}</ToolbarButton>
      </div>
    </div>
    {showForm && <div className='mb-4 grid grid-cols-2 gap-3 border-b border-[var(--shell-side-border)] pb-4 md:grid-cols-4'>
      <label className={FIELD}><span className={LABEL}>对象类型</span>
        <Dropdown value={entityKind} ariaLabel='对象类型' options={[{ value: 'FACILITY', label: '设施' }, { value: 'DEVICE', label: '设备' }]} onChange={(v) => { setEntityKind(v); setFacilityCode(''); setDeviceId('') }} /></label>
      {entityKind === 'FACILITY'
        ? <label className={FIELD}><span className={LABEL}>设施编码</span><Input value={facilityCode} onChange={(e) => setFacilityCode(e.target.value)} placeholder='P01001 / CLS00001' /></label>
        : <label className={FIELD}><span className={LABEL}>设备 ID</span><SimplePicker value={deviceId} onChange={setDeviceId} options={deviceOpts} ariaLabel={o.pickDevice} searchPlaceholder={o.pickDeviceSearch} minWidth={200} /></label>}
      <label className={FIELD}><span className={LABEL}>资产 ID(须 IN_STOCK/IN_TRANSIT)</span><SimplePicker value={assetId} onChange={setAssetId} search={searchAssets} ariaLabel={o.pickAsset} searchPlaceholder={o.pickAssetSearch} minWidth={220} /></label>
      <label className={FIELD}><span className={LABEL}>来源</span>
        <Dropdown value={sourceKind} ariaLabel='来源' options={[{ value: 'DIRECT', label: '直购直转' }, { value: 'PROCUREMENT', label: '采购入库' }, { value: 'CONSTRUCTION', label: '施工建成' }]} onChange={setSourceKind} /></label>
      {sourceKind === 'CONSTRUCTION' && <label className={FIELD}><span className={LABEL}>施工项目 ID(须已竣工)</span><SimplePicker value={projectId} onChange={setProjectId} options={projectOpts} ariaLabel={o.pickProject} searchPlaceholder={o.pickProjectSearch} minWidth={240} /></label>}
      <label className={FIELD}><span className={LABEL}>转固价值</span><Input value={valueAmount} onChange={(e) => setValueAmount(e.target.value)} placeholder='0.00' inputMode='decimal' /></label>
      <label className={FIELD}><span className={LABEL}>备注</span><Input value={remark} onChange={(e) => setRemark(e.target.value)} placeholder='可空' /></label>
      <div className='flex items-end'><ToolbarButton primary disabled={busy || !formOk} onClick={() => void register()}>登记</ToolbarButton></div>
    </div>}
    <div className='mb-2 flex flex-wrap items-center gap-2'>
      <Input className='w-40' value={reverseId} onChange={(e) => setReverseId(e.target.value)} placeholder='冲销凭证 ID' inputMode='numeric' />
      <Input value={reverseReason} onChange={(e) => setReverseReason(e.target.value)} placeholder='冲销原因(必填)' />
    </div>
    {rows.length === 0 ? <EmptyState text='暂无凭证' /> : <div className='overflow-x-auto'><Table>
      <TableHeader><TableRow><TableHead>凭证号</TableHead><TableHead>对象</TableHead><TableHead>资产 ID</TableHead><TableHead>来源</TableHead><TableHead>项目</TableHead><TableHead>批次</TableHead><TableHead>价值</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
      <TableBody>
        {rows.map((r) => <TableRow key={r.id}>
          <TableCell className='font-mono'>{r.registrationNo}</TableCell>
          <TableCell>{entityLabel(r)}</TableCell>
          <TableCell>{r.assetId}</TableCell>
          <TableCell>{SOURCE_TEXT[r.sourceKind] ?? r.sourceKind}</TableCell>
          <TableCell>{r.constructionProjectId > 0 ? r.constructionProjectId : '-'}</TableCell>
          <TableCell>{r.batchId > 0 ? r.batchId : '-'}</TableCell>
          <TableCell>{r.valueAmount.toFixed(2)}</TableCell>
          <TableCell><Badge variant={STATUS_VARIANT[r.status] ?? 'default'}>{STATUS_TEXT[r.status] ?? r.status}</Badge></TableCell>
          <TableCell>{r.status === 'ACTIVE'
            ? <ToolbarButton disabled={busy} onClick={() => void reverse(r.id)}>冲销</ToolbarButton>
            : <span className='text-xs opacity-60'>{r.reverseReason || '-'}</span>}</TableCell>
        </TableRow>)}
      </TableBody>
    </Table></div>}
  </section>
}
