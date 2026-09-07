// ROW 路权与 PECE 许可单列表页(P-INFRA-1 W4,000211;F3)。
// 设施/项目关联走 pickers 选择器(2026-09-07 域改造);新增文案走 pages.odn 三语词条。
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
import { PermitDetail } from './PermitDetail'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

export interface PermitRow {
  id: number
  permitNo: string
  kind: 'ROW' | 'PECE'
  title: string
  approvalNo: string
  authority: string
  validFrom?: string
  validUntil?: string
  status: string
  projectId: number
  projectNo: string
  facilityCode?: string
  chainId: number
  attachmentIds: number[]
  note: string
  rejectReason: string
  createdAt: string
}

export const PERMIT_KIND_TEXT: Record<string, string> = { ROW: 'ROW 路权', PECE: 'PECE 许可' }
export const PERMIT_STATUS_TEXT: Record<string, string> = {
  NOT_STARTED: '未开始', PENDING: '待处理', APPROVED: '已批准', EXPIRED: '已过期', NA: '不适用',
  PENDING_SIGN: '待签署', SIGNED: '已签署', STAMPED: '已盖章',
}
export const PERMIT_STATUS_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'danger'> = {
  NOT_STARTED: 'default', PENDING: 'warning', APPROVED: 'success', EXPIRED: 'danger', NA: 'default',
  PENDING_SIGN: 'warning', SIGNED: 'warning', STAMPED: 'success',
}

const ROW_STATUS_OPTIONS = ['NOT_STARTED', 'PENDING', 'APPROVED', 'EXPIRED', 'NA']
const PECE_STATUS_OPTIONS = ['PENDING_SIGN', 'SIGNED', 'STAMPED', 'NA']

// 选择器数据源行类型(ODN 本域列表接口;设施 /odn/facilities,项目 /odn/constructions)。
interface FacilityLite { code: string; name: string }
interface ProjectLite { id: number; projNo: string; name: string; status: string }

export default function PermitsPage() {
  const t = useT()
  const o = t.pages.odn
  const [rows, setRows] = useState<PermitRow[]>([])
  const [error, setError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [busy, setBusy] = useState(false)
  const [openId, setOpenId] = useState<number | null>(null)
  const [kind, setKind] = useState('')
  const [status, setStatus] = useState('')
  const [projectId, setProjectId] = useState('')
  const [form, setForm] = useState({ kind: 'ROW', title: '', approvalNo: '', authority: '', validFrom: '', validUntil: '', facilityCode: '', note: '' })
  const [facOpts, setFacOpts] = useState<DropdownOption[]>([])
  const [projectOpts, setProjectOpts] = useState<DropdownOption[]>([])

  // 设施/施工项目静态源一次拉取(空参=全网 ≤500/100 条,本地过滤;ODN 域同口径)。
  useEffect(() => {
    void (async () => {
      try {
        const facs = (await apiFetch<FacilityLite[]>('/odn/facilities')) ?? []
        setFacOpts(facs.map((x) => ({ value: x.code, label: x.code + (x.name ? ' ' + x.name : '') })))
      } catch { setFacOpts([]) }
      try {
        const projs = (await apiFetch<ProjectLite[]>('/odn/constructions', { query: { limit: 100 } })) ?? []
        setProjectOpts(projs.map((p) => ({ value: String(p.id), label: p.projNo + ' ' + (p.name || '') + ' [' + (o.projStatus[p.status as keyof typeof o.projStatus] ?? p.status) + ']' })))
      } catch { setProjectOpts([]) }
    })()
  }, [])

  const load = useCallback(async () => {
    setError('')
    try {
      const query: Record<string, string | number | undefined> = { limit: 200 }
      if (kind) query.kind = kind
      if (status) query.status = status
      if (projectId) query.projectId = Number(projectId)
      setRows((await apiFetch<PermitRow[]>('/odn/permits', { query })) ?? [])
    } catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [kind, status, projectId])

  useEffect(() => { void load() }, [load])

  const create = async () => {
    setBusy(true); setError('')
    try {
      await apiFetch('/odn/permits', { method: 'POST', body: { ...form, projectId: 0 } })
      toast.success('许可单已创建')
      setForm({ kind: 'ROW', title: '', approvalNo: '', authority: '', validFrom: '', validUntil: '', facilityCode: '', note: '' })
      setShowForm(false)
      await load()
    } catch (e) { setError(e instanceof Error ? e.message : '保存失败') } finally { setBusy(false) }
  }

  const statusOptions = kind === 'PECE' ? PECE_STATUS_OPTIONS : kind === 'ROW' ? ROW_STATUS_OPTIONS : []
  const set = (k: string, v: string) => setForm((m) => ({ ...m, [k]: v }))
  const field = (k: string, label: string, placeholder = '') => <label className={FIELD}><span className={LABEL}>{label}</span><Input value={form[k as keyof typeof form] ?? ''} placeholder={placeholder} onChange={(e) => set(k, e.target.value)} /></label>

  return <div>
    <div className='mb-3 flex items-center justify-between'><div className='flex flex-wrap items-end gap-2'>
      <Dropdown value={kind} ariaLabel='类型筛选' placeholder='全部类型' options={[{ value: 'ROW', label: 'ROW 路权' }, { value: 'PECE', label: 'PECE 许可' }]} onChange={(v) => { setKind(v); setStatus('') }} />
      {statusOptions.length > 0 && <Dropdown value={status} ariaLabel='状态筛选' placeholder='全部状态' options={statusOptions.map((s) => ({ value: s, label: PERMIT_STATUS_TEXT[s] }))} onChange={setStatus} />}
      <label className={FIELD}><span className={LABEL}>项目 ID</span><SimplePicker value={projectId} onChange={setProjectId} options={projectOpts} ariaLabel={o.pickProject} searchPlaceholder={o.pickProjectSearch} clearable clearLabel={t.pages.pickers.common.clear} minWidth={240} /></label>
    </div>
    <div className='flex items-center gap-2'><ToolbarButton primary onClick={() => setShowForm(!showForm)}>{showForm ? '取消' : '新建许可单'}</ToolbarButton><ToolbarButton onClick={() => void load()}>刷新</ToolbarButton></div></div>
    {showForm && <div className={CARD + ' mb-3 p-4'}><div className='grid grid-cols-2 gap-3 md:grid-cols-4'>
      <label className={FIELD}><span className={LABEL}>类型</span><Dropdown value={form.kind} ariaLabel='许可类型' options={[{ value: 'ROW', label: 'ROW 路权' }, { value: 'PECE', label: 'PECE 许可' }]} onChange={(v) => set('kind', v)} /></label>
      {field('title', '名称', '如 人民路架空段路权')}
      {field('approvalNo', '批复号', '批准前可留空')}
      {field('authority', '管辖机构', '如 市政公用局')}
      {field('validFrom', '有效期起', 'YYYY-MM-DD')}
      {field('validUntil', '有效期止', 'YYYY-MM-DD')}
      <label className={FIELD}><span className={LABEL}>关联设施</span><SimplePicker value={form.facilityCode} onChange={(v) => set('facilityCode', v)} options={facOpts} ariaLabel={o.pickFacility} searchPlaceholder={o.pickFacilitySearch} clearable clearLabel={t.pages.pickers.common.clear} minWidth={200} /></label>
      {field('note', '备注')}
    </div><div className='mt-3 flex justify-end'><ToolbarButton primary disabled={busy} onClick={() => void create()}>{busy ? '保存中…' : '保存'}</ToolbarButton></div></div>}
    {error && <ErrorBanner message={error} className='mb-3' />}
    <section className={CARD + ' overflow-hidden'}>
      {rows.length === 0 ? <EmptyState text='暂无许可单' /> : <div className='overflow-x-auto'><Table>
        <TableHeader><TableRow><TableHead>许可单号</TableHead><TableHead>类型</TableHead><TableHead>名称</TableHead><TableHead>批复号</TableHead><TableHead>管辖机构</TableHead><TableHead>有效期止</TableHead><TableHead>状态</TableHead><TableHead>关联项目</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map((r) => <TableRow key={r.id}>
            <TableCell className='font-mono'>{r.permitNo}</TableCell>
            <TableCell>{PERMIT_KIND_TEXT[r.kind] ?? r.kind}</TableCell>
            <TableCell>{r.title || '-'}</TableCell>
            <TableCell>{r.approvalNo || '-'}</TableCell>
            <TableCell>{r.authority || '-'}</TableCell>
            <TableCell>{r.validUntil || '-'}</TableCell>
            <TableCell><Badge variant={PERMIT_STATUS_VARIANT[r.status] ?? 'default'}>{PERMIT_STATUS_TEXT[r.status] ?? r.status}</Badge></TableCell>
            <TableCell>{r.projectNo ? <span className='font-mono'>{r.projectNo}</span> : <span className='text-xs opacity-60'>未关联</span>}</TableCell>
            <TableCell><button className='text-[var(--color-text-link)]' onClick={() => setOpenId(openId === r.id ? null : r.id)}>{openId === r.id ? '收起' : '详情'}</button></TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </section>
    {openId != null && <PermitDetail permitId={openId} onChanged={() => void load()} />}
  </div>
}