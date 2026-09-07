// ODN 点位反查抽屉(T14-1):GIS 页 ODN 图层点位点击 → 资源信息/关联资源/业务绑定/就近可装性。
// 反馈链路:开启即骨架加载;任一接口失败显原因(可复制)+重试;无绑定显空态;成功分区渲染。
import { useEffect, useState } from 'react'
import { Drawer } from '../../../components/Drawer'
import { Badge } from '../../../components/ui/badge'
import { CopyButton, Spinner } from '../../../components/business/feedback'
import { useT } from '../../../i18n'
import type { GisPoint } from '../../../components/business/maps'
import { loadOdnReverse, type OdnReverseData } from './odnReverse'

type Phase = { s: 'loading' } | { s: 'ok'; d: OdnReverseData } | { s: 'fail'; msg: string }
type Gis = ReturnType<typeof useT>['pages']['gisPage']

// 状态枚举 → 中文文案(禁裸状态码);未知值回退原码不臆造。
function statusLabel(g: Gis, s: string): string {
  const map: Record<string, string> = {
    IN_USE: g.revStatusInUse, ACTIVE: g.revStatusActive, RETIRED: g.revStatusRetired,
    IDLE: g.revStatusIdle, RESERVED: g.revStatusReserved, IN_SERVICE: g.revStatusInService,
  }
  return map[s] ?? s
}

function lifecycleLabel(g: Gis, s: string): string {
  const map: Record<string, string> = {
    PLANNED: g.revLifePlanned, IN_BUILD: g.revLifeInBuild, IN_SERVICE: g.revLifeInService, RETIRED: g.revLifeRetired,
  }
  return map[s] ?? s
}

// 就近可装性三态徽标:SERVED 绿 / PENDING 蓝 / UNSERVED 红(与覆盖关联页同语义)。
function ServeBadge({ g, status }: { g: Gis; status: string }) {
  const v = status === 'SERVED' ? 'success' : status === 'PENDING' ? 'info' : 'danger'
  const label = status === 'SERVED' ? g.revServeOk : status === 'PENDING' ? g.revServePend : status === 'UNSERVED' ? g.revServeNo : status
  return <Badge variant={v}>{label}</Badge>
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-1.5">
      <span className="shrink-0 text-[13px] text-[var(--shell-group-title)]">{k}</span>
      <span className="min-w-0 truncate text-right text-[13px] text-[var(--shell-content-text)]" title={v}>{v}</span>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mb-4">
      <h4 className="mb-1 mt-0 border-b border-[var(--shell-side-border)] pb-1 text-[13px] font-semibold text-[var(--shell-heading)]">{title}</h4>
      {children}
    </section>
  )
}

const Muted = ({ children }: { children: React.ReactNode }) => (
  <div className="py-2 text-[13px] text-[var(--shell-group-title)]">{children}</div>
)

export function OdnReverseDrawer({ point, onClose }: { point: GisPoint | null; onClose: () => void }) {
  const t = useT()
  const g = t.pages.gisPage
  const [phase, setPhase] = useState<Phase>({ s: 'loading' })

  useEffect(() => {
    if (!point) return
    let alive = true
    setPhase({ s: 'loading' })
    loadOdnReverse(point)
      .then((d) => { if (alive) setPhase({ s: 'ok', d }) })
      .catch((e) => { if (alive) setPhase({ s: 'fail', msg: e instanceof Error ? e.message : g.revLoadFail }) })
    return () => { alive = false }
  }, [point]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!point) return null
  const retry = () => {
    setPhase({ s: 'loading' })
    loadOdnReverse(point)
      .then((d) => setPhase({ s: 'ok', d }))
      .catch((e) => setPhase({ s: 'fail', msg: e instanceof Error ? e.message : g.revLoadFail }))
  }
  const kindLabel = (d: OdnReverseData) => {
    const fac = g.revFacKind as Record<string, string>
    const dev = g.revDevKind as Record<string, string>
    return d.entity === 'facility' ? (fac[d.kind] ?? d.kind)
      : d.entity === 'device' ? (dev[d.kind] ?? d.kind)
        : g.revSiteType
  }
  return (
    <Drawer title={g.revTitle} onClose={onClose}>
      {phase.s === 'loading' && (
        <div className="flex items-center gap-2 py-8 text-[13px] text-[var(--shell-group-title)]" role="status">
          <Spinner /> {t.common.loading}
        </div>
      )}
      {phase.s === 'fail' && (
        <div>
          <div className="mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">
            {g.revLoadFail}: {phase.msg}
          </div>
          <div className="flex items-center gap-2">
            <CopyButton text={phase.msg} />
            <button
              className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]"
              onClick={retry}
            >{g.revRetry}</button>
          </div>
        </div>
      )}
      {phase.s === 'ok' && (
        <div>
          <Section title={g.detailTitle}>
            <Row k={g.detailFields[0]} v={phase.d.code} />
            <Row k={g.detailFields[1]} v={phase.d.name} />
            <Row k={g.detailFields[2]} v={kindLabel(phase.d)} />
            <Row k={g.detailFields[3]} v={statusLabel(g, phase.d.status)} />
            <Row k={g.revLifecycle} v={lifecycleLabel(g, phase.d.lifecycle)} />
            <Row k={g.revCoords} v={phase.d.lat.toFixed(5) + ', ' + phase.d.lng.toFixed(5)} />
          </Section>
          <Section title={g.odnLayerTitle}>
            {phase.d.gridName ? <Row k={g.revGrid} v={phase.d.gridName} /> : null}
            {phase.d.siteName ? <Row k={g.revSite} v={phase.d.siteName} /> : null}
            {phase.d.parentName ? <Row k={g.revParentDev} v={phase.d.parentName} /> : null}
            {!phase.d.gridName && !phase.d.siteName && !phase.d.parentName ? <Muted>{g.empty}</Muted> : null}
          </Section>
          <Section title={g.revBindings}>
            {phase.d.bindings.length === 0 ? <Muted>{g.revBindEmpty}</Muted> : phase.d.bindings.map((b, i) => (
              <div key={i} className="flex flex-wrap items-baseline justify-between gap-x-3 py-1.5">
                <span className="text-[13px] text-[var(--shell-content-text)]">{g.revPort} {b.portLabel}</span>
                <span className="text-[13px] text-[var(--shell-group-title)]">{statusLabel(g, b.portStatus)}</span>
                <span className="text-[13px] text-[var(--shell-content-text)]">{g.revOrder} #{b.orderId}</span>
                <span className="text-[12px] text-[var(--shell-group-title)]">{g.revBoundAt} {b.boundAt}</span>
              </div>
            ))}
          </Section>
          <Section title={g.revServability}>
            {phase.d.coverage ? (
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center gap-2">
                  <ServeBadge g={g} status={phase.d.coverage.status} />
                  {/* resolve 契约 distanceM omitempty:点位即最近设施时距离 0 被省略,按 0m 呈现 */}
                  {phase.d.coverage.facilityCode ? (
                    <span className="text-[13px] text-[var(--shell-content-text)]">{t.pages.odn.distance}: {Math.round(phase.d.coverage.distanceM ?? 0)}m</span>
                  ) : null}
                </div>
                {phase.d.coverage.facilityCode ? (
                  <Row k={g.revNearest} v={phase.d.coverage.facilityCode + ' ' + (phase.d.coverage.facilityName ?? '')} />
                ) : null}
              </div>
            ) : (
              <Muted>{g.empty}</Muted>
            )}
          </Section>
        </div>
      )}
    </Drawer>
  )
}