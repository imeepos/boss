// 用户详情抽屉:GET /users/{customerId} 14 段聚合的分层展示(主档头部 + 概览统计 + 页签分区)。
// 文案/颜色走 i18n 与主题令牌;枚举渲染对齐 StatusTag 注册域与 terms.md 枚举。
import { useEffect, useState, type ReactNode } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { TabBar } from '../../../components/business/tab-bar'
import { EmptyState } from '../../../components/business/page-head'
import { fmtFee, fmtTime } from '../../../lib/format'
import type { UserRow } from './filter'
import { DETAIL_TABS, NOTIFY_CARD_KEY, PROFILE_FIELDS, SECTION_LIMIT, latestPlanName,
  type ColSpec, type DetailCol, type DetailSection } from './detail-view'

type Detail = Record<string, unknown>

const cellBase = 'px-3 py-2 text-xs align-top border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)]'

function renderCell(v: unknown, spec: ColSpec | undefined, d: Record<string, string>): ReactNode {
  if (v === null || v === undefined || v === '') return <span className="text-[var(--shell-crumb-text)]">—</span>
  switch (spec?.kind) {
    case 'time': return fmtTime(String(v))
    case 'fee': return fmtFee(Number(v))
    case 'feeCents': return fmtFee(Number(v) / 100)
    case 'bool': return v ? d.dYes : d.dNo
    case 'tag': return <StatusTag domain={spec.domain} value={String(v)} />
    case 'enum': return d[spec.map[String(v)] ?? ''] ?? String(v)
    default: return String(v)
  }
}

function SectionBlock({ section, detail, empty }: {
  section: DetailSection
  detail: Detail
  empty: string
}) {
  const t = useT()
  const u = t.pages.userPage
  const d = u.d
  const [expanded, setExpanded] = useState(false)
  if (section.key === NOTIFY_CARD_KEY) return <NotifyPrefsCard detail={detail} />
  const rows = (detail[section.key] as Record<string, unknown>[] | undefined) ?? []
  const limit = section.limit ?? SECTION_LIMIT
  const capped = rows.length > limit
  const visible = expanded ? rows : rows.slice(0, limit)
  return (
    <section className="mb-5">
      <h4 className="mb-2 mt-0 flex items-center gap-2 text-[13px] font-semibold text-[var(--shell-heading)]">
        {u.sectionNames[section.key] ?? section.key}
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
                    {section.cols.map((c: DetailCol) => (
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

const PREF_ITEMS = [
  { k: 'business', labelKey: 'dNotifyBusiness' },
  { k: 'marketing', labelKey: 'dNotifyMarketing' },
] as const

function NotifyPrefsCard({ detail }: { detail: Detail }) {
  const t = useT()
  const prefs = (detail.notify ?? {}) as Record<string, unknown>
  return (
    <section className="mb-5">
      <h4 className="mb-2 mt-0 text-[13px] font-semibold text-[var(--shell-heading)]">{t.pages.userPage.sectionNames.notify}</h4>
      <div className="flex flex-wrap gap-x-8 gap-y-2 rounded-sm border border-[var(--shell-card-border)] px-3 py-3 text-xs text-[var(--shell-content-text)]">
        {PREF_ITEMS.map(({ k, labelKey }) => (
          <span key={k}>
            {t.pages.userPage.d[labelKey]}:
            <b className="ml-1 font-medium">{prefs[k] ? t.pages.userPage.d.dOn : t.pages.userPage.d.dOff}</b>
          </span>
        ))}
        <span>
          {t.pages.userPage.d.dNotifyChannel}:<b className="ml-1 font-medium">{typeof prefs.channel === 'string' && prefs.channel ? prefs.channel : '—'}</b>
        </span>
      </div>
    </section>
  )
}

function StatChip({ label, value, danger }: { label: string; value: ReactNode; danger?: boolean }) {
  return (
    <div className="min-w-0 flex-1 rounded-sm border border-[var(--shell-card-border)] px-3 py-2" style={danger ? { borderColor: 'color-mix(in srgb, var(--color-danger) 35%, transparent)' } : undefined}>
      <div className="text-[11px] text-[var(--shell-group-title)]">{label}</div>
      <div className="truncate text-[13px] font-medium" style={danger ? { color: 'var(--color-danger)' } : undefined}>{value}</div>
    </div>
  )
}

export function UserDetailDrawer({ id, summary, onClose }: {
  id: number
  /** 列表行摘要:加载完成前先渲染已知字段。 */
  summary?: UserRow
  onClose: () => void
}) {
  const t = useT()
  const u = t.pages.userPage
  const [detail, setDetail] = useState<Detail | null>(null)
  const [error, setError] = useState('')
  const [tab, setTab] = useState(DETAIL_TABS[0].key)

  useEffect(() => {
    apiFetch<Detail>(`/users/${id}`)
      .then(setDetail)
      .catch((e) => setError(e instanceof Error ? e.message : u.loadFail))
  }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  const name = typeof detail?.name === 'string' && detail.name ? detail.name : summary?.name ?? ''
  const balances = Array.isArray(detail?.balances) ? (detail.balances as Record<string, unknown>[]) : []
  const wallet = balances[0]
  const planName = latestPlanName(detail?.plans)

  return (
    <Drawer title={`${u.detailTitle} #${id}`} onClose={onClose} width={720}
      footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{u.back}</button>}>
      {error ? (
        <div className="rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
      ) : !detail ? (
        <EmptyState text={u.loading} />
      ) : (
        <div>
          {/* 主档头部 */}
          <div className="flex items-start gap-3">
            <span className="flex h-11 w-11 flex-none items-center justify-center rounded-full bg-[var(--shell-menu-hover-bg)] text-base font-semibold text-[var(--shell-heading)]">{name.slice(0, 1) || '?'}</span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-base font-semibold text-[var(--shell-heading)]">{name || '—'}</span>
                {typeof detail.serviceStatus === 'string' && detail.serviceStatus && <StatusTag domain="service" value={detail.serviceStatus} />}
                {typeof detail.realNameStatus === 'string' && detail.realNameStatus && <StatusTag domain="realName" value={detail.realNameStatus} />}
              </div>
              <div className="mt-1 grid grid-cols-2 gap-x-6 gap-y-1 text-xs text-[var(--shell-content-text)]">
                {PROFILE_FIELDS.map((f) => (
                  <span key={f} className="truncate">
                    <span className="text-[var(--shell-crumb-text)]">{u.d[`d_${f}`]} </span>
                    {String(detail[f] || '—')}
                  </span>
                ))}
              </div>
            </div>
          </div>
          {/* 概览统计 */}
          <div className="my-4 flex gap-2">
            <StatChip label={u.d.dStatBalance} value={wallet ? fmtFee(Number(wallet.balance)) : '—'} danger={Boolean(wallet?.lowWarn)} />
            <StatChip label={u.d.dStatPlan} value={planName || '—'} />
            <StatChip label={u.d.dStatOrders} value={(detail.orders as unknown[] | undefined)?.length ?? 0} />
            <StatChip label={u.d.dStatCoupons} value={(detail.coupons as unknown[] | undefined)?.length ?? 0} />
          </div>
          <TabBar tabs={DETAIL_TABS.map((x) => ({ key: x.key, label: u.detailTabs[x.key] }))} value={tab} onChange={setTab} />
          {DETAIL_TABS.filter((x) => x.key === tab).map((x) => (
            <div key={x.key}>
              {x.sections.map((s) => <SectionBlock key={s.key} section={s} detail={detail} empty={u.empty} />)}
            </div>
          ))}
        </div>
      )}
    </Drawer>
  )
}
