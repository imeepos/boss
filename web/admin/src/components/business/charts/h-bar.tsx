// 横向条形:N 项排名,每行 label + 进度条 + 数值;max 自动归一。
// 适用 ROI 排名、利用率等"绝对值排名"场景。

export function HorizontalBar({
  items, format,
}: {
  items: Array<{ label: string; value: number }>
  format?: (v: number) => string
}) {
  const max = Math.max(1, ...items.map((x) => x.value))
  const fmt = format ?? ((v) => v.toFixed(2))
  return (
    <ul className="flex flex-col gap-2 py-2">
      {items.map((it, i) => (
        <li key={it.label + i} className="flex items-center gap-3">
          <span className="w-32 flex-none truncate text-[13px] text-[var(--shell-content-text)]" title={it.label}>{it.label}</span>
          <div className="h-2 flex-1 overflow-hidden rounded-sm bg-[var(--color-surface-disabled)]">
            <div className="h-full rounded-sm bg-[var(--color-text-link)] transition-[width] duration-300" style={{ width: `${(it.value / max) * 100}%` }} />
          </div>
          <span className="w-20 flex-none text-right text-[13px] tabular-nums text-[var(--shell-content-text)]">{fmt(it.value)}</span>
        </li>
      ))}
    </ul>
  )
}