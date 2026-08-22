// 统计卡:label + value + delta(趋势色)。源自 dashboard,图表组件化首例。
import type { ReactNode } from 'react'

const TREND_CLASS = {
  up: 'text-[var(--color-danger)]',
  down: 'text-[var(--color-success)]',
  flat: 'text-muted-foreground',
} as const

export type Trend = keyof typeof TREND_CLASS

export function StatCard({
  label, value, delta, trend, icon, onClick,
}: {
  label: string
  value: ReactNode
  delta?: string
  trend?: Trend
  icon?: ReactNode
  onClick?: () => void
}) {
  const interactive = Boolean(onClick)
  const activate = () => onClick?.()

  return (
    <div
      className={'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]' + (interactive ? ' cursor-pointer transition-colors hover:bg-[var(--shell-menu-hover-bg)]' : '')}
      onClick={activate}
      onKeyDown={(e) => {
        if (interactive && (e.key === 'Enter' || e.key === ' ')) {
          e.preventDefault()
          activate()
        }
      }}
      tabIndex={interactive ? 0 : undefined}
      role={interactive ? 'link' : undefined}
    >
      <div className="mb-2 flex items-center justify-between text-sm text-[var(--shell-content-text)]">
        <span>{label}</span>
        {icon && <span className="text-[var(--color-text-link)]">{icon}</span>}
      </div>
      <div className="text-[28px] font-semibold leading-tight text-[var(--shell-heading)]">{value}</div>
      {delta && <div className={'mt-1 text-[13px] ' + (TREND_CLASS[trend ?? 'flat'])}>{delta}</div>}
    </div>
  )
}