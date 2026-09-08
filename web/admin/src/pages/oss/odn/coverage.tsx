import { toast } from 'sonner'
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { fmtTime } from '../../../lib/format'
import { CoverageRegisterDrawer } from './CoverageRegisterDrawer'

type Cov = { id: number; addressId: number; facilityCode: string; deviceId: number; status: string; note: string; addressName?: string; updatedAt: string }
type Resolved = { status: string; facilityCode?: string; facilityName?: string; distanceM?: number }
type Opt = { value: string; label: string }

// StatusBadge 可装状态徽标:SERVED 绿 / PENDING 蓝 / UNSERVED 红。
function StatusBadge({ status, labels }: { status: string; labels: Record<string, string> }) {
  const variant = status === 'SERVED' ? 'success' : status === 'PENDING' ? 'info' : 'danger'
  return <Badge variant={variant}>{labels[status] ?? status}</Badge>
}

// CoveragePanel 覆盖关联页签(可查可判;后端 /odn/coverage*)。
// 关联对象一律选择器:地址=服务端检索,设施/设备=本城市主数据静态源(路线图规则 3)。
export function CoveragePanel({ g, prv, city }: { g: any; prv: string; city: string }) {
  const [rows, setRows] = useState<Cov[]>([])
  const [error, setError] = useState('')
  const [showRegister, setShowRegister] = useState(false)
  const [ll, setLl] = useState({ lat: '', lng: '' })
  const [resolved, setResolved] = useState<Resolved | null>(null)
  const [facOpts, setFacOpts] = useState<Opt[]>([])
  const [devOpts, setDevOpts] = useState<Opt[]>([])
  const badgeLabels: Record<string, string> = { SERVED: g.covServed, PENDING: g.covPending, UNSERVED: g.covUnserved }

  // 本城市设施/设备主数据:覆盖表单选择器数据源(空城市不拉)。
  const loadRefs = useCallback(async () => {
    if (!prv || !city) return
    try {
      const facs = await apiFetch<{ code: string; name: string | null }[]>('/odn/facilities', { query: { prvCode: prv, cityPrefix: city } }) ?? []
      setFacOpts(facs.map((x) => ({ value: x.code, label: x.code + (x.name ? ' ' + x.name : '') })))
      const devs = await apiFetch<{ id: number; code: string; name: string | null }[]>('/odn/devices', { query: { prvCode: prv, cityPrefix: city } }) ?? []
      setDevOpts(devs.map((x) => ({ value: String(x.id), label: x.code + (x.name ? ' ' + x.name : '') })))
    } catch { setFacOpts([]); setDevOpts([]) }
  }, [city, prv])
  useEffect(() => { void loadRefs() }, [loadRefs])

  const searchAddresses = async (kw: string) => {
    const hits = await apiFetch<{ node: { id: number; name: string } }[]>('/addresses/search', { query: { q: kw } }) ?? []
    return hits.map((h) => ({ value: String(h.node.id), label: h.node.name }))
  }

  const load = useCallback(async () => {
    setError('')
    try { setRows(await apiFetch<Cov[]>('/odn/coverage/list', { query: { limit: 200 } }) ?? []) }
    catch (e) {
      const msg = e instanceof Error ? e.message : g.loadFail
      setError(msg)
      toast.error(g.loadFail, { description: msg })
    }
  }, [g.loadFail])
  useEffect(() => { void load() }, [load])

  const resolve = async () => {
    setError(''); setResolved(null)
    try { setResolved(await apiFetch<Resolved>('/odn/coverage/resolve', { query: { lat: ll.lat, lng: ll.lng } })) }
    catch (e) {
      const msg = e instanceof Error ? e.message : g.loadFail
      setError(msg)
      toast.error(g.loadFail, { description: msg })
    }
  }

  return <div>
    {error && <ErrorBanner message={error} className="mb-3" />}
    <div className="mb-3 flex justify-end">
      <ToolbarButton primary onClick={() => setShowRegister(true)}>登记覆盖关联</ToolbarButton>
    </div>
    {showRegister && <CoverageRegisterDrawer facOpts={facOpts} devOpts={devOpts} searchAddresses={searchAddresses}
      onClose={() => setShowRegister(false)} onCreated={() => void load()} />}
    <div className="flex flex-wrap items-end gap-2 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]">
      <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">{g.lat}</span><Input value={ll.lat} placeholder="14.5995" onChange={(e) => setLl((v) => ({ ...v, lat: e.target.value }))} /></label>
      <label className="flex flex-col gap-1"><span className="text-xs text-[var(--shell-content-text)]">{g.lng}</span><Input value={ll.lng} placeholder="120.9842" onChange={(e) => setLl((v) => ({ ...v, lng: e.target.value }))} /></label>
      <ToolbarButton disabled={!ll.lat || !ll.lng} onClick={resolve}>{g.resolveBtn}</ToolbarButton>
      {resolved && <span className="flex items-center gap-2 text-sm text-[var(--shell-content-text)]">
        <StatusBadge status={resolved.status} labels={badgeLabels} />
        {resolved.facilityCode && <span className="font-mono">{resolved.facilityCode} {resolved.facilityName}</span>}
        {resolved.distanceM != null && <span>{g.distance}: {Math.round(resolved.distanceM)}m</span>}
      </span>}
    </div>
    <section className="mt-3 overflow-hidden rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      {rows.length === 0 ? <EmptyState text={g.empty} /> : <div className="overflow-x-auto"><Table>
        <TableHeader><TableRow>{[g.addressId, g.addressName, g.covFacility, g.covDevice, g.covStatus, g.covNote, g.updatedAt].map((h) => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader>
        <TableBody>{rows.map((r) => <TableRow key={r.id}>
          <TableCell>{r.addressId}</TableCell>
          <TableCell>{r.addressName || '-'}</TableCell>
          <TableCell className="font-mono">{r.facilityCode || '-'}</TableCell>
          <TableCell>{r.deviceId || '-'}</TableCell>
          <TableCell><StatusBadge status={r.status} labels={badgeLabels} /></TableCell>
          <TableCell>{r.note || '-'}</TableCell>
          <TableCell>{fmtTime(r.updatedAt)}</TableCell>
        </TableRow>)}</TableBody>
      </Table></div>}
    </section>
  </div>
}