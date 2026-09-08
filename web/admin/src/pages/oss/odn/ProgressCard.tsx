// W7 施工进度卡(F5a):进度上报留痕列表+清单级聚合(挂施工单详情)。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner } from '../../../components/business/page-head'

interface ProgressEntry {
  id: number
  facilityCode: string
  doneQty: number
  lat: number
  lng: number
  note: string
  photoIds: number[]
  reporterType: string
  reporterName?: string
  reportedAt: string
}
interface ItemProgress { facilityCode: string; plannedQty: number; doneQty: number; entries: number }

const reporterLabel = (e: ProgressEntry) => e.reporterType === 'WORKER'
  ? '师傅 ' + (e.reporterName || '')
  : '账号 ' + (e.reporterName || '');

export function ProgressCard({ projectId }: { projectId: number }) {
  const [entries, setEntries] = useState<ProgressEntry[]>([])
  const [items, setItems] = useState<ItemProgress[]>([])
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setError('')
    try {
      const d = await apiFetch<{ entries: ProgressEntry[]; items: ItemProgress[] }>('/odn/constructions/' + projectId + '/progress')
      setEntries(d?.entries ?? []);
      setItems(d?.items ?? []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
      toast.error('进度上报加载失败', { description: msg })
    }
  }, [projectId])
  useEffect(() => { void load() }, [load])

  return <Card className="p-4">
    <div className="mb-2 flex items-center justify-between">
      <div className="text-sm font-semibold">施工进度上报<span className="ml-2 text-xs font-normal opacity-60">只增不改留痕;上报人限管理账号或师傅(W7)</span></div>
      <button className="text-xs text-[var(--color-text-link)]" onClick={() => void load()}>刷新</button>
    </div>
    {error && <ErrorBanner message={error} className='mb-2' />}
    {items.length > 0 && <div className='mb-3 flex flex-wrap gap-2'>
      {items.map((it) => <Badge key={it.facilityCode} variant={it.doneQty >= it.plannedQty && it.plannedQty > 0 ? 'success' : 'default'}>
        {it.facilityCode} {it.doneQty}/{it.plannedQty}
      </Badge>)}
    </div>}
    {entries.length === 0 ? <EmptyState text='暂无进度上报' /> : <div className='overflow-x-auto'><Table>
      <TableHeader><TableRow><TableHead>设施</TableHead><TableHead>完成量</TableHead><TableHead>坐标</TableHead><TableHead>备注</TableHead><TableHead>照片</TableHead><TableHead>上报人</TableHead><TableHead>时间</TableHead></TableRow></TableHeader>
      <TableBody>
        {entries.map((e) => <TableRow key={e.id}>
          <TableCell className='font-mono'>{e.facilityCode}</TableCell>
          <TableCell>{e.doneQty}</TableCell>
          <TableCell className='font-mono text-xs'>{e.lat ? e.lat.toFixed(5) + ',' + e.lng.toFixed(5) : '-'}</TableCell>
          <TableCell className='max-w-[220px] truncate'>{e.note || '-'}</TableCell>
          <TableCell>{e.photoIds.length > 0 ? 'x' + e.photoIds.length : '-'}</TableCell>
          <TableCell>{reporterLabel(e)}</TableCell>
          <TableCell>{e.reportedAt}</TableCell>
        </TableRow>)}
      </TableBody>
    </Table></div>}
  </Card>
}