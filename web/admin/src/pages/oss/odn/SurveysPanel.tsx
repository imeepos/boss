// W7 勘测任务面板(挂 ODN 管理页勘测页签;P-INFRA-1 W7,迁移 000223)。
// 文案为字面量:同 constructions 面板口径,不动 i18n 中央登记。
// 新建走右侧抽屉 SurveyCreateDrawer(2026-09-09):与全站表单口径统一,不再用页内内联卡片。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Badge } from '../../../components/ui/badge'
import { Dropdown } from '../../../components/Dropdown'
import { Drawer } from '../../../components/Drawer'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { SurveyCreateDrawer, type WorkerLite } from './SurveyCreateDrawer'

interface SurveyTask {
  id: number
  taskNo: string
  title: string
  description: string
  prvCode: string
  cityPrefix: string
  gridCode: number
  assignedWorkerId: number
  workerName?: string
  status: string
  reportCount: number
  createdAt: string
}
interface SurveyReport {
  id: number
  workerId: number
  workerName?: string
  lat: number
  lng: number
  facilityNote: string
  suggestion: string
  photoIds: number[]
  reportedAt: string
}

const STATUS_TEXT: Record<string, string> = { PENDING: '待执行', ACCEPTED: '已接单', BACKFILLED: '已回填', CANCELLED: '已取消' }
const STATUS_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'danger' | 'info'> = { PENDING: 'warning', ACCEPTED: 'info', BACKFILLED: 'success', CANCELLED: 'danger' }
const SUGGEST_TEXT: Record<string, string> = { CAN_INSTALL: '可装', NEED_NEW_FACILITY: '需新建设施' }

export default function SurveysPanel() {
  const [rows, setRows] = useState<SurveyTask[]>([])
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [workers, setWorkers] = useState<WorkerLite[]>([])
  const [detail, setDetail] = useState<SurveyTask | null>(null)
  const [reports, setReports] = useState<SurveyReport[]>([])

  const load = useCallback(async () => {
    setError('')
    try {
      const xs = await apiFetch<{ items: SurveyTask[] }>('/odn/surveys', { query: { limit: 200 } })
      setRows(xs?.items ?? [])
    } catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [])

  useEffect(() => { void load() }, [load])
  useEffect(() => {
    void (async () => {
      try {
        const xs = await apiFetch<{ items: WorkerLite[] }>('/workers', { query: { limit: 500 } })
        setWorkers((xs?.items ?? []).filter((w) => w.id > 0))
      } catch { setWorkers([]) }
    })()
  }, [])

  const assign = async (t: SurveyTask, workerId: number) => {
    try {
      await apiFetch('/odn/surveys/' + t.id + '/assign', { method: 'POST', body: { workerId } })
      toast.success('已指派 ' + (workers.find((w) => w.id === workerId)?.name ?? workerId))
      await load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '指派失败'
      setError(msg)
      toast.error('指派失败', { description: msg })
    }
  }

  const cancel = async (t: SurveyTask) => {
    try {
      await apiFetch('/odn/surveys/' + t.id + '/cancel', { method: 'POST' })
      toast.success('任务已取消')
      await load()
    } catch (e) {
      const msg = e instanceof Error ? e.message : '取消失败'
      setError(msg)
      toast.error('取消失败', { description: msg })
    }
  }

  const openDetail = async (t: SurveyTask) => {
    setDetail(t)
    try {
      const d = await apiFetch<{ task: SurveyTask; reports: SurveyReport[] }>('/odn/surveys/' + t.id)
      setReports(d?.reports ?? [])
    } catch { setReports([]) }
  }

  const assignable = (t: SurveyTask) => t.status === 'PENDING' || t.status === 'ACCEPTED'

  return <div>
    <div className='mb-3 flex items-center justify-between'>
      <ToolbarButton primary onClick={() => setShowCreate(true)}>新建勘测任务</ToolbarButton>
      <ToolbarButton onClick={() => void load()}>刷新</ToolbarButton>
    </div>
    {showCreate && <SurveyCreateDrawer workers={workers} onClose={() => setShowCreate(false)} onCreated={() => void load()} />}
    {error && <ErrorBanner message={error} className='mb-3' />}
    {rows.length === 0 ? <EmptyState text='暂无勘测任务' /> : <div className='overflow-x-auto'><Table>
      <TableHeader><TableRow><TableHead>任务号</TableHead><TableHead>标题</TableHead><TableHead>目标网格</TableHead><TableHead>指派师傅</TableHead><TableHead>状态</TableHead><TableHead>回填数</TableHead><TableHead>创建时间</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
      <TableBody>
        {rows.map((t) => <TableRow key={t.id}>
          <TableCell className='font-mono'>{t.taskNo}</TableCell>
          <TableCell>{t.title}</TableCell>
          <TableCell>{t.gridCode > 0 ? String(t.gridCode).padStart(2, '0') : '-'}</TableCell>
          <TableCell>{t.workerName || (t.assignedWorkerId > 0 ? '#' + t.assignedWorkerId : '抢单池')}</TableCell>
          <TableCell><Badge variant={STATUS_VARIANT[t.status] ?? 'default'}>{STATUS_TEXT[t.status] ?? t.status}</Badge></TableCell>
          <TableCell>{t.reportCount}</TableCell>
          <TableCell>{t.createdAt}</TableCell>
          <TableCell className='whitespace-nowrap'>
            <button className='mr-2 text-[var(--color-text-link)]' onClick={() => void openDetail(t)}>详情</button>
            {assignable(t) && <span className='mr-2'>
              <Dropdown value='' ariaLabel={'指派 ' + t.taskNo} placeholder='指派/改派' searchable searchPlaceholder='搜索师傅'
                options={workers.map((w) => ({ value: String(w.id), label: w.name }))}
                onChange={(v) => { if (v) void assign(t, Number(v)) }} /></span>}
            {assignable(t) && <button className='text-[var(--color-text-link)]' onClick={() => void cancel(t)}>取消</button>}
          </TableCell>
        </TableRow>)}
      </TableBody>
    </Table></div>}
    {detail && <Drawer title={detail.taskNo + ' ' + detail.title} onClose={() => setDetail(null)}>
      {detail.description && <p className='mb-3 text-xs opacity-70'>{detail.description}</p>}
      {reports.length === 0 ? <EmptyState text='暂无回填记录' /> : <div className='space-y-2'>
        {reports.map((r) => <div key={r.id} className='rounded-md border border-[var(--shell-card-border)] p-3'>
          <div className='mb-1 flex flex-wrap items-center gap-2'>
            <Badge variant={r.suggestion === 'CAN_INSTALL' ? 'success' : 'warning'}>{SUGGEST_TEXT[r.suggestion] ?? r.suggestion}</Badge>
            <span className='text-xs'>{r.workerName || '#' + r.workerId}</span>
            <span className='text-xs opacity-60'>{r.reportedAt}</span>
          </div>
          {r.facilityNote && <div className='text-xs'>{r.facilityNote}</div>}
          <div className='mt-1 font-mono text-xs opacity-60'>{r.lat ? r.lat.toFixed(6) + ', ' + r.lng.toFixed(6) : '无打点'}{r.photoIds.length > 0 ? ' | 照片x' + r.photoIds.length : ''}</div>
        </div>)}
      </div>}
    </Drawer>}
  </div>
}