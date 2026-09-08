// 师傅选择器:指派/转派抽屉内检索并选择师傅,展示工号/联系方式/班组/负责区域/评分/已接单。
// 数据源:/workers /worker-groups /regions /worker-performances(无数据或 403 时降级)、
// /dispatch/my-tickets?workerId=(仅对可见行逐个取已接单数)。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Input } from '../../../components/ui/input'

interface WorkerRow {
  id: number
  staffNo: string
  name: string
  groupId: number
  groupName: string // 班组名(服务端 JOIN 现值,data-relations §6.2)
  regionId: number
  regionName: string // 主区域名(服务端 JOIN 现值,data-relations §6.2)
  phone: string
  status: number // 1 在职 0 离职
}

interface GroupRow { id: number; name: string }
interface RegionRow { id: number; name: string }
interface PerfRow { workerId: number; period: string; score: number; finished: number; onTimeRate: number }
interface TicketRow { status: string }

export interface PickedWorker extends WorkerRow {
  groupName: string
  regionName: string
  score: number | null
}

const CARD = 'flex items-center gap-3 rounded-sm border px-3 py-2.5 text-left cursor-pointer'
const CARD_IDLE = 'border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] hover:border-[var(--color-border-focus)] hover:bg-[var(--shell-menu-hover-bg)]'
const CARD_ON = 'border-[var(--color-border-focus)] bg-[var(--shell-menu-hover-bg)]'
const LABEL = 'text-xs text-[var(--shell-crumb-text)]'
const VALUE = 'text-[13px] text-[var(--shell-heading)]'

export function WorkerPicker({ selectedId, onSelect }: {
  selectedId: string
  onSelect: (w: PickedWorker) => void
}) {
  const t = useT()
  const p = t.pages.dispatchPage.picker
  const [workers, setWorkers] = useState<WorkerRow[]>([])
  const [groups, setGroups] = useState<Map<number, string>>(new Map())
  const [regions, setRegions] = useState<Map<number, string>>(new Map())
  const [perfs, setPerfs] = useState<Map<number, PerfRow>>(new Map())
  const [loads, setLoads] = useState<Map<number, { total: number; doing: number }>>(new Map())
  const [keyword, setKeyword] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<{ items: WorkerRow[] }>('/workers')
      .then((d) => setWorkers(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
    apiFetch<{ items: GroupRow[] }>('/worker-groups')
      .then((d) => setGroups(new Map((d?.items ?? []).map((g) => [g.id, g.name]))))
      .catch(() => { /* 行内 groupName 已直供,映射仅兜底 */ })
    // /regions 返回裸数组(无 items 包裹),兼容两种形态。
    apiFetch<RegionRow[]>('/regions')
      .then((d) => {
        const rows = Array.isArray(d) ? d : ((d as unknown as { items?: RegionRow[] })?.items ?? [])
        setRegions(new Map(rows.map((r) => [r.id, r.name])))
      })
      .catch(() => { /* 行内 regionName 已直供,映射仅兜底 */ })
    // 评分取每位师傅最新 period 一行;无数据/无权限(403)时静默降级为 '—'。
    apiFetch<{ items: PerfRow[] }>('/worker-performances')
      .then((d) => {
        const m = new Map<number, PerfRow>()
        for (const r of d?.items ?? []) {
          const prev = m.get(r.workerId)
          if (!prev || r.period > prev.period) m.set(r.workerId, r)
        }
        setPerfs(m)
      })
      .catch(() => { /* 评分降级 */ })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const kw = keyword.trim().toLowerCase()
  const filtered = useMemo(() =>
    workers.filter((w) =>
      !kw || w.name.toLowerCase().includes(kw)
      || w.staffNo.toLowerCase().includes(kw)
      || w.phone.includes(kw)), [workers, kw])

  // 已接单数:仅对检索结果前 20 条逐个查询,避免全量扫。
  useEffect(() => {
    for (const w of filtered.slice(0, 20)) {
      if (loads.has(w.id)) continue
      apiFetch<{ items: TicketRow[] }>('/dispatch/my-tickets', { query: { workerId: w.id } })
        .then((d) => {
          const items = d?.items ?? []
          setLoads((prev) => new Map(prev).set(w.id, {
            total: items.length,
            doing: items.filter((x) => x.status === 'DOING').length,
          }))
        })
        .catch(() => { /* 已接单降级为 '—' */ })
    }
  }, [filtered, loads]) // eslint-disable-line react-hooks/exhaustive-deps

  const pick = (w: WorkerRow): PickedWorker => ({
    ...w,
    // 行内 JOIN 名优先,列表映射兜底;都取不到降级 —,不再露 #id(§6.2)。
    groupName: w.groupName || groups.get(w.groupId) || '—',
    regionName: w.regionName || regions.get(w.regionId) || '—',
    score: perfs.get(w.id)?.score ?? null,
  })

  return (
    <div className="flex flex-col gap-2">
      <Input className="h-8 w-full" value={keyword} placeholder={p.search} onChange={(e) => setKeyword(e.target.value)} />
      {error && <div className="text-xs text-[var(--color-danger)]">{error}</div>}
      <div className="flex max-h-72 flex-col gap-2 overflow-y-auto">
        {filtered.map((w) => {
          const perf = perfs.get(w.id)
          const load = loads.get(w.id)
          const on = String(w.id) === selectedId
          // 离职师傅后端 workerAssignable 拒收:置灰禁选,免得必失败的提交。
          const off = w.status !== 1
          return (
            <button
              key={w.id}
              className={`${CARD} ${on ? CARD_ON : CARD_IDLE} ${off ? 'cursor-not-allowed opacity-50' : ''}`}
              disabled={off}
              onClick={() => onSelect(pick(w))}
            >
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-baseline gap-x-2">
                  <span className={'text-[13px] font-semibold ' + (w.status === 1 ? '' : 'opacity-60')}>{w.name}</span>
                  <span className={LABEL}>{w.staffNo}</span>
                  {w.status !== 1 && <span className="text-xs text-[var(--color-danger)]">{p.left}</span>}
                </div>
                <div className="mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
                  <span className={LABEL}>{p.phone}: <span className={VALUE}>{w.phone || '—'}</span></span>
                  <span className={LABEL}>{p.group}: <span className={VALUE}>{w.groupName || groups.get(w.groupId) || '—'}</span></span>
                  <span className={LABEL}>{p.region}: <span className={VALUE}>{w.regionName || regions.get(w.regionId) || '—'}</span></span>
                  <span className={LABEL}>{p.score}: <span className={VALUE}>{perf ? perf.score.toFixed(1) : '—'}</span></span>
                  <span className={LABEL}>{p.accepted}: <span className={VALUE}>{load ? `${load.total}(${p.doing} ${load.doing})` : '—'}</span></span>
                </div>
              </div>
              <span className="flex-shrink-0 text-xs text-primary">{on ? p.selected : p.select}</span>
            </button>
          )
        })}
        {!filtered.length && <div className="py-6 text-center text-[13px] text-[var(--shell-group-title)]">{p.noMatch}</div>}
      </div>
    </div>
  )
}
