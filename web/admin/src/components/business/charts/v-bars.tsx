// 纵向柱状:双列 N 日/N 项趋势,SVG/CSS 实现(无依赖)。源自 dashboard 7 日趋势柱。
// valueLabel? 后缀(如 "%"/"单")便于 value 美化;emptyText 控制空态文案。
export function VerticalBars({
  labels, values, valueLabel, emptyText = '暂无数据',
}: {
  labels: string[]
  values: number[]
  /** 柱顶数字后缀(如 "%" 显示 "50%") */
  valueLabel?: string
  /** 空态文案(labels.length===0 时显示) */
  emptyText?: string
}) {
  if (labels.length === 0 || values.length === 0) {
    return <div className="flex h-40 items-center justify-center text-[13px] text-[var(--shell-group-title)]">{emptyText}</div>
  }
  const max = Math.max(1, ...values)
  return (
    <div className="flex h-40 items-end gap-3 overflow-x-auto py-5">
      {labels.map((lab, i) => (
        <div key={lab + i} className="flex h-full min-w-12 flex-col items-center justify-end gap-2" title={`${lab}: ${values[i]}${valueLabel ?? ''}`}>
          <div className="text-xs font-medium text-muted-foreground">{values[i]}{valueLabel ?? ''}</div>
          <div className="min-h-1 w-3/5 max-w-8 rounded-t-sm bg-primary transition-[height] duration-300" style={{ height: `${(values[i] / max) * 100}%` }} />
          <span className="text-xs text-muted-foreground">{lab}</span>
        </div>
      ))}
    </div>
  )
}