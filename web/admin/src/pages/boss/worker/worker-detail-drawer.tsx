// 师傅详情抽屉:GET /workers/{workerId} 主档头 + 概览芯片 + 关联子集段
// (工单/消息/绩效/评价,各取一段,超限折叠)。文案/颜色走 i18n 与主题令牌;
// 枚举渲染对齐 StatusTag 注册域与 terms.md §4。追加式实现,不新增全站通用组件。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { EmptyState, LoadingState } from '../../../components/business'
import { fmtTime } from '../../../lib/format'
import {
  ACTIVE_TICKET_STATUSES, SECTION_LIMIT, WORKER_DETAIL_SECTIONS, WORKER_PROFILE_FIELDS,
  WORKER_STATUS_KEY, type WColSpec, type WDetailSection,
} from './worker-detail-view'

type WorkerDetail = Record<string, unknown>

const cellBase = 'px-3 py-2 text-xs align-top border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'

function renderCell(v: unknown, spec: WColSpec | undefined, d: Record<string, string>): React.ReactNode {
  if (v === null || v === undefined || v === '') return <span className="text-[var(--shell-crumb-text)]">—</span>
  switch (spec?.kind) {
    case 'time': return fmtTime(String(v))
    case 'tag': return <StatusTag domain={spec.domain} value={String(v)} />
    case 'enum': return d[spec.map[String(v)] ?? ''] ?? String(v)
    default: return String(v)
  }
}

function SectionBlock({ section, rows, name, d, empty }: {
  section: WDetailSection
  rows: Record<string, unknown>[]
  name: string
  d: Record<string, string>
  empty: string
}) {
  const [expanded, setExpanded] = useState(false)
  const limit = section.limit || SECTION_LIMIT
  const capped = rows.length > limit
  const visible = expanded ? rows : rows.slice(0, limit)
  return (
    <section className="mb-5">
      <h4 className="mb-2 mt-0 flex items-center gap-2 text-[13px] font-semibold text-[var(--shell-heading)]">
        {name}
        <span className="rounded-full bg-[var(--shell-menu-hover-bg)] px-2 text-[11px] leading-4 text-[var(--shell-group-title)]">{rows.length}</span>
      </h4>
      {rows.length === 0 ? <EmptyState text={empty} /> : (
        <>
          <div className="overflow-x-auto rounded-sm border border-[var(--shell-card-border)]">
            <table className="w-full border-collapse">
              <thead>
                <tr>
                  {section.cols.map((c) => (
                    <th key={c.key} className="h-9 bg-[var(--shell-menu-hover-bg)] px-3 text-left text-[11px] font-medium whitespace-nowrap text-[var(--shell-group-title)]">{d[c.k] ?? c.key}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {visible.map((row, i) => (
                  <tr key={i} className="hover:bg-[var(--shell-menu-hover-bg)]">
                    {section.cols.map((c) => (
                      <td key={c.key} className={cellBase}>{renderCell(row[c.key], c.spec, d)}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {capped && (
            <button className="mt-2 cursor-pointer border-none bg-none px-0 text-xs text-[var(--color-border-focus)] hover:underline" onClick={() => setExpanded(!expanded)}>
              {expanded ? d.dCollapse : d.dSeeAll.replace('{count}', String(rows.length))}
            </button>
          )}
        </>
      )}
    </section>
  )
}

function StatChip({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="min-w-0 flex-1 rounded-sm border border-[var(--shell-card-border)] px-3 py-2">
      <div className="text-[11px] text-[var(--shell-group-title)]">{label}</div>
      <div className="truncate text-[13px] font-medium text-[var(--shell-heading)]">{value}</div>
    </div>
  )
}

/** id → name 映射(尽力而为:regions 尚需 menu:region 权限,失败降级回退 id 展示)。 */
async function nameMap(url: string): Promise<Record<number, string>> {
  try {
    const data = await apiFetch<{ items: Array<Record<string, unknown>> } | Array<Record<string, unknown>>>(url)
    const list = Array.isArray(data) ? data : (data?.items ?? [])
    const m: Record<number, string> = {}
    list.forEach((x) => {
      if (typeof x.id === 'number' && typeof x.name === 'string') m[x.id as number] = x.name
    })
    return m
  } catch {
    return {}
  }
}

export function WorkerDetailDrawer({ id, groupName, onClose, onResetPwd }: {
  id: number
  /** 列表行已知班组名,主档渲染前先展示。 */
  groupName?: string
  onClose: () => void
  /** 提供即展示"重置密码"(师傅端登录密码,WorkerDialogs 承接)。 */
  onResetPwd?: (workerId: number, name: string) => void
}) {
  const t = useT()
  const w = t.pages.workerPage
  const d = w.d
  const [detail, setDetail] = useState<WorkerDetail | null>(null)
  const [sub, setSub] = useState<Record<string, Record<string, unknown>[]>>({})
  const [regionNames, setRegionNames] = useState<Record<number, string>>({})
  const [groupNames, setGroupNames] = useState<Record<number, string>>({})
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    setDetail(null)
    setSub({})
    setError('')
    apiFetch<WorkerDetail>(`/workers/${id}`)
      .then((data) => { if (alive) setDetail(data) })
      .catch((e) => { if (alive) setError(e instanceof Error ? e.message : w.loadFail) })
    void nameMap('/worker-groups').then((m) => { if (alive) setGroupNames(m) })
    void nameMap('/regions').then((m) => { if (alive) setRegionNames(m) })
    Promise.allSettled(
      WORKER_DETAIL_SECTIONS.map((s) =>
        apiFetch<{ items: Record<string, unknown>[] }>(s.api, { query: { workerId: String(id) } })
          .then((data) => [s.key, data?.items ?? []] as const),
      ),
    ).then((results) => {
      if (!alive) return
      const next: Record<string, Record<string, unknown>[]> = {}
      results.forEach((r) => { if (r.status === 'fulfilled') next[r.value[0]] = r.value[1] })
      setSub(next)
    })
    return () => { alive = false }
  }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  const rowsFor = (key: string) => sub[key] ?? []
  const tickets = rowsFor('tickets')
  const activeTickets = useMemo(() =>
    tickets.filter((tk) => ACTIVE_TICKET_STATUSES.includes(String(tk.status) as (typeof ACTIVE_TICKET_STATUSES)[number])).length, [tickets])

  const statusLabel = (s: number) => (WORKER_STATUS_KEY[s] ? w[WORKER_STATUS_KEY[s] as 'active' | 'left'] : '')

  const name = typeof detail?.name === 'string' && detail.name ? detail.name : ''
  const status = typeof detail?.status === 'number' ? (detail.status as number) : 1
  const fmt = (v: unknown) => (v === null || v === undefined || v === '' ? '—' : String(v))

  // 主档展示值:region/group 尽力转名,失败降级回退 id;时间为格式化。
  const fieldValue = (key: string, v: unknown) => {
    if (key === 'regionId') return regionNames[Number(v)] ?? fmt(v)
    if (key === 'groupId') return groupNames[Number(v)] ?? (groupName && Number(v) === Number(detail?.groupId) ? groupName : fmt(v))
    if (key === 'joinedAt' || key === 'leftAt') return v ? fmtTime(String(v)) : '—'
    return fmt(v)
  }

  return (
    <Drawer title={`${w.detailTitle} #${id}`} onClose={onClose} width={720}
      footer={
        <div className="flex justify-end gap-2">
          {onResetPwd && status === 1 && (
            <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]"
              onClick={() => onResetPwd(id, name)}>{w.resetPwd}</button>
          )}
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{w.cancel}</button>
        </div>
      }>
      {error ? (
        <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
      ) : !detail ? (
        <LoadingState />
      ) : (
        <div>
          {/* 主档头部 */}
          <div className="flex items-start gap-3">
            <span className="flex h-11 w-11 flex-none items-center justify-center rounded-full bg-[var(--shell-menu-hover-bg)] text-base font-semibold text-[var(--shell-heading)]">{name.slice(0, 1) || '?'}</span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-base font-semibold text-[var(--shell-heading)]">{name || '—'}</span>
                <span className="text-xs text-[var(--shell-crumb-text)]">{fmt(detail.staffNo)}</span>
                {WORKER_STATUS_KEY[status] && (
                  <span className="rounded-sm bg-[var(--shell-menu-hover-bg)] px-2 py-0.5 text-xs text-[var(--shell-group-title)]">{statusLabel(status)}</span>
                )}
              </div>
              <div className="mt-1 grid grid-cols-2 gap-x-6 gap-y-1 text-xs text-[var(--shell-content-text)]">
                {WORKER_PROFILE_FIELDS.map((f) => (
                  <span key={f.key} className="truncate">
                    <span className="text-[var(--shell-crumb-text)]">{d[f.labelKey]}</span>{' '}
                    {fieldValue(f.key, detail[f.key])}
                  </span>
                ))}
              </div>
            </div>
          </div>
          {/* 概览统计 */}
          <div className="my-4 flex gap-2">
            <StatChip label={d.dStatTickets} value={activeTickets} />
            <StatChip label={d.dStatTotalTickets} value={tickets.length} />
            <StatChip label={d.dStatMessages} value={rowsFor('messages').length} />
            <StatChip label={d.dStatFeedbacks} value={rowsFor('feedbacks').length} />
          </div>
          {/* 关联子集段(工单/消息/绩效/评价):各取一段,超限折叠 */}
          {WORKER_DETAIL_SECTIONS.map((s) => (
            <SectionBlock key={s.key} section={s} rows={rowsFor(s.key)} name={w.sectionNames[s.key as 'tickets' | 'messages' | 'performances' | 'feedbacks']} d={d} empty={w.empty} />
          ))}
        </div>
      )}
    </Drawer>
  )
}