// 堆叠柱状:N 柱(横向)× K 段堆叠;每柱总高度=各段之和,max 归一。
// 用于"四周期对比(柱并列,柱内堆叠指标)"场景;非趋势图。

const PALETTE = ['#C69835', '#1F355F', '#BE8D25', '#273F70', '#9CAAC3']

export interface StackedBarSeries {
  /** 每柱组的多段值(柱内堆叠的指标值) */
  stacks: number[]
}

export function StackedBars({
  groups, legends,
}: {
  /** 每柱一组数据,K 段堆叠。groups 与柱数对齐,length=柱数。 */
  groups: StackedBarSeries[]
  /** K 段名称(图例) */
  legends: string[]
}) {
  if (groups.length === 0) return null
  const totals = groups.map((g) => g.stacks.reduce((s, v) => s + Math.max(0, v), 0))
  const max = Math.max(1, ...totals)
  return (
    <div className="flex flex-col gap-3">
      <div className="flex h-48 items-end gap-4">
        {groups.map((g, i) => {
          const total = totals[i] ?? 0
          const h = total > 0 ? (total / max) * 100 : 0
          return (
            <div key={i} className="flex h-full w-12 flex-col items-center justify-end gap-1.5" title={`${total}`}>
              <div className="text-[11px] text-[var(--shell-group-title)]">{total}</div>
              <div className="flex w-full flex-col-reverse overflow-hidden rounded-t-sm bg-[var(--color-surface-disabled)]" style={{ height: `${h}%` }}>
                {g.stacks.map((v, k) => {
                  const segH = total > 0 ? (Math.max(0, v) / total) * 100 : 0
                  return <div key={k} className="w-full" style={{ height: `${segH}%`, background: PALETTE[k % PALETTE.length] }} title={`${legends[k] ?? k}: ${v}`} />
                })}
              </div>
            </div>
          )
        })}
      </div>
      <ul className="flex flex-wrap items-center gap-3 text-[12px]">
        {legends.map((name, i) => (
          <li key={name} className="flex items-center gap-1.5">
            <span className="inline-block h-3 w-3 rounded-sm" style={{ background: PALETTE[i % PALETTE.length] }} />
            <span className="text-[var(--shell-content-text)]">{name}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}