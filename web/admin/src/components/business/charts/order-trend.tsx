import { useEffect, useMemo, useState } from 'react'
import {
  Area, CartesianGrid, Line, LineChart, LabelList,
  ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts'

interface OrderTrendPoint { label: string; value: number }
interface OrderTrendProps {
  labels: string[]
  values: number[]
  valueUnit: string
  tooltipLabel: string
  emptyText: string
  interactionLabels: { previous: string; next: string; zoomOut: string; zoomIn: string; reset: string }
}

const MIN_VISIBLE = 5

function formatValue(value: number, unit: string): string { return `${value}${unit}` }
function points(labels: string[], values: number[]): OrderTrendPoint[] {
  return labels.map((label, index) => ({ label, value: values[index] ?? 0 }))
}
function clamp(value: number, min: number, max: number): number { return Math.max(min, Math.min(max, value)) }

export function OrderTrend({ labels, values, valueUnit, tooltipLabel, emptyText, interactionLabels }: OrderTrendProps) {
  const data = useMemo(() => points(labels, values), [labels, values])
  const [range, setRange] = useState({ start: 0, end: Math.max(0, data.length - 1) })
  const [dragStart, setDragStart] = useState<number | null>(null)
  const visible = data.slice(range.start, range.end + 1)
  useEffect(() => {
    setRange({ start: 0, end: Math.max(0, data.length - 1) })
  }, [data.length])
  const canPan = data.length > MIN_VISIBLE
  const setWindow = (start: number) => {
    const size = range.end - range.start
    const nextStart = clamp(start, 0, Math.max(0, data.length - size))
    setRange({ start: nextStart, end: nextStart + size })
  }
  const moveBy = (delta: number) => setWindow(range.start + delta)
  const zoom = (factor: number) => {
    const current = range.end - range.start + 1
    const next = clamp(Math.round(current * factor), MIN_VISIBLE, data.length)
    const center = Math.round((range.start + range.end) / 2)
    const start = clamp(center - Math.floor(next / 2), 0, data.length - next)
    setRange({ start, end: start + next - 1 })
  }
  const reset = () => setRange({ start: 0, end: Math.max(0, data.length - 1) })

  if (data.length === 0) {
    return <div className="flex h-64 items-center justify-center text-[13px] text-[var(--shell-group-title)]">{emptyText}</div>
  }

  return (
    <div className="flex w-full flex-col gap-3" role="img" aria-label={tooltipLabel}>
      <div
        className={`h-80 w-full${canPan ? ' cursor-grab select-none active:cursor-grabbing' : ''}`}
        onPointerDown={(event) => {
          if (!canPan) return
          event.currentTarget.setPointerCapture(event.pointerId)
          const rect = event.currentTarget.getBoundingClientRect()
          setDragStart(Math.round((event.clientX - rect.left) / rect.width * Math.max(1, visible.length - 1)) + range.start)
        }}
        onPointerMove={(event) => {
          if (dragStart === null) return
          const rect = event.currentTarget.getBoundingClientRect()
          const index = Math.round((event.clientX - rect.left) / rect.width * Math.max(1, visible.length - 1)) + range.start
          moveBy(dragStart - index)
          setDragStart(index)
        }}
        onPointerUp={() => setDragStart(null)}
        onPointerCancel={() => setDragStart(null)}
      >
        <ResponsiveContainer width="100%" height="100%" minWidth={280}>
          <LineChart data={visible} margin={{ top: 20, right: 12, left: -12, bottom: 4 }}>
            <defs>
              <linearGradient id="order-trend-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="var(--color-brand-gold-500)" stopOpacity={0.28} />
                <stop offset="100%" stopColor="var(--color-brand-gold-500)" stopOpacity={0.02} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} stroke="var(--shell-side-border)" strokeDasharray="3 5" />
            <XAxis dataKey="label" interval={visible.length > 14 ? Math.ceil(visible.length / 12) - 1 : 0} minTickGap={16} axisLine={false} tickLine={false} tick={{ fill: 'var(--shell-group-title)', fontSize: 11 }} dy={8} />
            <YAxis allowDecimals={false} axisLine={false} tickLine={false} width={30} tick={{ fill: 'var(--shell-group-title)', fontSize: 11 }} />
            <Tooltip
              cursor={{ stroke: 'var(--color-brand-gold-500)', strokeDasharray: '4 4', strokeOpacity: 0.65 }}
              contentStyle={{ background: 'var(--shell-card-bg)', border: '1px solid var(--shell-card-border)', borderRadius: '8px', boxShadow: 'var(--shell-card-shadow)', color: 'var(--shell-heading)', fontSize: '12px' }}
              labelStyle={{ color: 'var(--shell-heading)', fontWeight: 600, marginBottom: 4 }}
              formatter={(value) => [formatValue(Number(value), valueUnit), tooltipLabel]}
            />
            <Area type="monotone" dataKey="value" stroke="none" fill="url(#order-trend-fill)" />
            <Line type="monotone" dataKey="value" stroke="var(--color-brand-gold-500)" strokeWidth={3} dot={{ r: 5, fill: 'var(--shell-card-bg)', stroke: 'var(--color-brand-gold-500)', strokeWidth: 2 }} activeDot={{ r: 7, fill: 'var(--color-brand-gold-500)', stroke: 'var(--shell-card-bg)', strokeWidth: 2 }} connectNulls>
              <LabelList dataKey="value" position="top" offset={10} fill="var(--shell-heading)" fontSize={12} fontWeight={600} formatter={(value: number) => formatValue(value, valueUnit)} />
            </Line>
          </LineChart>
        </ResponsiveContainer>
      </div>
      {canPan && (
        <div className="flex items-center justify-between gap-2">
          <button type="button" aria-label={interactionLabels.previous} className="h-7 rounded border border-[var(--shell-side-border)] px-2 text-xs text-[var(--shell-content-text)] disabled:opacity-40" disabled={range.start === 0} onClick={() => moveBy(-Math.max(1, Math.floor((range.end - range.start + 1) / 3)))}>←</button>
          <div className="relative h-2 flex-1 rounded-full bg-[var(--shell-menu-hover-bg)]" aria-label="trend range">
            <div className="absolute h-full rounded-full bg-[var(--color-brand-gold-500)]" style={{ left: `${range.start / data.length * 100}%`, right: `${(data.length - range.end - 1) / data.length * 100}%` }} />
          </div>
          <button type="button" aria-label={interactionLabels.next} className="h-7 rounded border border-[var(--shell-side-border)] px-2 text-xs text-[var(--shell-content-text)] disabled:opacity-40" disabled={range.end === data.length - 1} onClick={() => moveBy(Math.max(1, Math.floor((range.end - range.start + 1) / 3)))}>→</button>
          <button type="button" aria-label={interactionLabels.zoomOut} className="h-7 rounded border border-[var(--shell-side-border)] px-2 text-xs text-[var(--shell-content-text)]" onClick={() => zoom(0.75)}>−</button>
          <button type="button" aria-label={interactionLabels.zoomIn} className="h-7 rounded border border-[var(--shell-side-border)] px-2 text-xs text-[var(--shell-content-text)]" onClick={() => zoom(1.25)}>＋</button>
          <button type="button" aria-label={interactionLabels.reset} className="h-7 rounded border border-[var(--shell-side-border)] px-2 text-xs text-[var(--shell-content-text)]" onClick={reset}>Reset</button>
        </div>
      )}
    </div>
  )
}
