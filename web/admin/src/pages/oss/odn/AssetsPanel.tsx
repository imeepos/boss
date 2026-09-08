// ODN 资产化转固面板(P-INFRA-1 W8):凭证列表/登记/冲销 + 出库台账入口。
// 关联实体一律 pickers 选择器(2026-09-07 域改造);新增文案走 pages.odn 三语词条。
// 登记/冲销走右侧抽屉(2026-09-09):与全站表单口径统一,不再有页内内联写入表单。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Badge } from '../../../components/ui/badge'
import { Dropdown } from '../../../components/Dropdown'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { AssetRegisterDrawer } from './AssetRegisterDrawer'
import { AssetReverseDrawer } from './AssetReverseDrawer'

interface Registration {
  id: number; registrationNo: string; entityKind: string; facilityCode: string; deviceId: number
  assetId: number; sourceKind: string; constructionProjectId: number; batchId: number
  valueAmount: number; status: string; reverseReason: string; remark: string
  registeredBy: number; registeredAt: string
}

const SOURCE_TEXT: Record<string, string> = { PROCUREMENT: '采购入库', CONSTRUCTION: '施工建成', DIRECT: '直购直转' }

const STATUS_VARIANT: Record<string, 'info' | 'success' | 'danger'> = { ACTIVE: 'success', REVERSED: 'danger' }
const STATUS_TEXT: Record<string, string> = { ACTIVE: '有效', REVERSED: '已冲销' }

function entityLabel(r: Registration): string {
  return r.entityKind === 'FACILITY' ? (r.facilityCode || '-') : '设备#' + r.deviceId
}

export function AssetsPanel() {
  const [rows, setRows] = useState<Registration[]>([])
  const [status, setStatus] = useState('ACTIVE')
  const [error, setError] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [reverseTarget, setReverseTarget] = useState<Registration | null>(null)

  const load = useCallback(async () => {
    setError('')
    try {
      const q: Record<string, string | number | undefined> = { limit: 200 }
      if (status) q.status = status
      setRows((await apiFetch<Registration[]>('/odn/assets/registrations', { query: q })) ?? [])
    } catch (e) {
      const msg = e instanceof Error ? e.message : '加载失败'
      setError(msg)
      toast.error('凭证加载失败', { description: msg })
    }
  }, [status])
  useEffect(() => { void load() }, [load])

  return <section className="mt-4"><Card className="p-4">
    {error && <ErrorBanner message={error} className="mb-3" />}
    <div className="mb-3 flex flex-wrap items-center gap-2">
      <span className="text-sm font-semibold">资产化凭证</span>
      <span className="text-xs opacity-60">施工建成设施/设备(含导入域箱体)凭证据此获得资产身份;价值与采购/项目溯源随凭证登记</span>
      <div className="ml-auto flex items-end gap-2">
        <Dropdown value={status} ariaLabel="凭证状态" options={[{ value: 'ACTIVE', label: '有效' }, { value: 'REVERSED', label: '已冲销' }, { value: '', label: '全部' }]} onChange={setStatus} />
        <ToolbarButton primary onClick={() => setShowCreate(true)}>资产化登记</ToolbarButton>
      </div>
    </div>
    {showCreate && <AssetRegisterDrawer onClose={() => setShowCreate(false)} onCreated={() => void load()} />}
    {reverseTarget && <AssetReverseDrawer id={reverseTarget.id} registrationNo={reverseTarget.registrationNo}
      onClose={() => setReverseTarget(null)} onReversed={() => void load()} />}
    {rows.length === 0 ? <EmptyState text="暂无凭证" /> : <div className="overflow-x-auto"><Table>
      <TableHeader><TableRow><TableHead>凭证号</TableHead><TableHead>对象</TableHead><TableHead>资产 ID</TableHead><TableHead>来源</TableHead><TableHead>项目</TableHead><TableHead>批次</TableHead><TableHead>价值</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
      <TableBody>
        {rows.map((r) => <TableRow key={r.id}>
          <TableCell className="font-mono">{r.registrationNo}</TableCell>
          <TableCell>{entityLabel(r)}</TableCell>
          <TableCell>{r.assetId}</TableCell>
          <TableCell>{SOURCE_TEXT[r.sourceKind] ?? r.sourceKind}</TableCell>
          <TableCell>{r.constructionProjectId > 0 ? r.constructionProjectId : '-'}</TableCell>
          <TableCell>{r.batchId > 0 ? r.batchId : '-'}</TableCell>
          <TableCell>{r.valueAmount.toFixed(2)}</TableCell>
          <TableCell><Badge variant={STATUS_VARIANT[r.status] ?? 'default'}>{STATUS_TEXT[r.status] ?? r.status}</Badge></TableCell>
          <TableCell>{r.status === 'ACTIVE'
            ? <ToolbarButton onClick={() => setReverseTarget(r)}>冲销</ToolbarButton>
            : <span className="text-xs opacity-60">{r.reverseReason || '-'}</span>}</TableCell>
        </TableRow>)}
      </TableBody>
    </Table></div>}
  </Card></section>
}