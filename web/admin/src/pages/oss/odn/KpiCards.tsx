// ODN 关联统计卡(spec oss-odn-v2 §3):网格/设施/局点/关联 OLT;第四卡蓝色
// tinted 图标强调跨域关联(资源域 /resources)。计数取页面已加载行数。
import { Building2, LayoutGrid, MapPin, Server } from 'lucide-react'

export interface KpiTexts { grids: string; facilities: string; sites: string; linkedOlt: string; capacity: string }

interface KpiProps {
  grids: number
  facilities: number
  sites: number
  olts: number
  /** 最高网格占用 %(facilities/999 取最大),空城市不渲染容量条。 */
  capacityPct: number
  g: KpiTexts
}

const CARD = 'flex-1 min-w-[180px] rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]'
const ICON_BOX = 'flex h-10 w-10 items-center justify-center rounded-md'

// tinted 图标底:color-mix 14% alpha + 同源文字色(主题变量,双主题自适应)。
const TONES: Record<string, { box: string; icon: string }> = {
  navy: { box: 'background: color-mix(in srgb, var(--color-brand-blue-700) 14%, transparent)', icon: 'text-[var(--color-brand-blue-700)]' },
  green: { box: 'background: color-mix(in srgb, var(--color-success) 14%, transparent)', icon: 'text-[var(--color-success)]' },
  blue: { box: 'background: color-mix(in srgb, var(--color-info) 14%, transparent)', icon: 'text-[var(--color-info)]' },
}

export function KpiCards({ grids, facilities, sites, olts, capacityPct, g }: KpiProps) {
  const cards = [
    { label: g.grids, value: grids, icon: LayoutGrid, tone: 'navy' },
    { label: g.facilities, value: facilities, icon: MapPin, tone: 'green' },
    { label: g.sites, value: sites, icon: Building2, tone: 'navy' },
    { label: g.linkedOlt, value: olts, icon: Server, tone: 'blue' },
  ]
  return <div className="flex flex-wrap items-stretch gap-4">
    {cards.map((c) => {
      const tone = TONES[c.tone]
      const Icon = c.icon
      return <div key={c.label} className={CARD}>
        <div className="flex items-center gap-3">
          <span className={ICON_BOX} style={{ background: tone.box }}><Icon className={'h-5 w-5 ' + tone.icon} /></span>
          <div className="min-w-0">
            <div className="truncate text-xs text-[var(--color-text-secondary)]">{c.label}</div>
            <div className="text-2xl font-semibold leading-7 text-[var(--shell-heading)]">{c.value}</div>
          </div>
        </div>
        {c.label === g.grids && grids > 0 && (
          <div className="mt-3 flex items-center gap-2">
            <span className="text-[11px] text-[var(--color-text-tertiary)]">{g.capacity}</span>
            <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--shell-menu-hover-bg)]">
              <span className="block h-full rounded-full bg-[var(--color-brand-blue-700)]" style={{ width: Math.min(100, capacityPct) + '%' }} />
            </span>
            <span className="text-[11px] text-[var(--color-text-tertiary)]">{capacityPct}%</span>
          </div>
        )}
      </div>
    })}
  </div>
}