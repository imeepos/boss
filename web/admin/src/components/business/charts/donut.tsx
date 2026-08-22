// 占比环形:多段 N 等分,SVG <circle stroke-dasharray> 实现,中心显示总数。
// 颜色按 --color-text-link 金 → --color-brand-navy-950 深蓝 → 灰阶 6 段。

const PALETTE = ['#C69835', '#1F355F', '#BE8D25', '#273F70', '#9CAAC3', '#A6B1C3']

export interface DonutSegment { label: string; value: number }

export function Donut({
  segments, total,
}: {
  segments: DonutSegment[]
  total?: number // 显式给定时覆盖 sum,避免精度问题
}) {
  const sum = total ?? segments.reduce((s, x) => s + Math.max(0, x.value), 0)
  if (sum <= 0 || segments.length === 0) {
    return <div className="flex h-40 items-center justify-center text-[13px] text-[var(--shell-group-title)]">—</div>
  }
  const r = 50
  const c = 2 * Math.PI * r
  let acc = 0
  return (
    <div className="flex items-center gap-4 py-2">
      <svg width="140" height="140" viewBox="0 0 120 120" role="img" aria-label="占比环形">
        <g transform="rotate(-90 60 60)">
          {segments.map((s, i) => {
            const v = Math.max(0, s.value)
            const len = (v / sum) * c
            const dash = `${len} ${c - len}`
            const offset = -(acc / sum) * c
            acc += v
            return (
              <circle key={s.label + i} cx="60" cy="60" r={r} fill="none"
                stroke={PALETTE[i % PALETTE.length]} strokeWidth="18"
                strokeDasharray={dash} strokeDashoffset={offset} />
            )
          })}
        </g>
        <text x="60" y="58" textAnchor="middle" className="fill-[var(--shell-heading)]" fontSize="20" fontWeight="600">{sum}</text>
        <text x="60" y="76" textAnchor="middle" className="fill-[var(--shell-group-title)]" fontSize="11">总数</text>
      </svg>
      <ul className="flex flex-col gap-1.5 text-[13px]">
        {segments.map((s, i) => (
          <li key={s.label} className="flex items-center gap-2">
            <span className="inline-block h-3 w-3 rounded-sm" style={{ background: PALETTE[i % PALETTE.length] }} />
            <span className="text-[var(--shell-content-text)]">{s.label}</span>
            <span className="text-[var(--shell-group-title)]">· {s.value} ({((s.value / sum) * 100).toFixed(1)}%)</span>
          </li>
        ))}
      </ul>
    </div>
  )
}